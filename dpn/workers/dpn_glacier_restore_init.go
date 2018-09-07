package workers

import (
	"fmt"
	"github.com/APTrust/exchange/constants"
	"github.com/APTrust/exchange/context"
	"github.com/APTrust/exchange/dpn/models"
	dpn_network "github.com/APTrust/exchange/dpn/network"
	apt_network "github.com/APTrust/exchange/network"
	"github.com/nsqio/go-nsq"
	"strconv"
	"strings"
	"time"
)

// Standard retrieval is 3-5 hours.
// Bulk is 5-12 hours, and is cheaper.
// There's no rush on DPN fixity checking, so use the cheaper option.
// https://docs.aws.amazon.com/amazonglacier/latest/dev/downloading-an-archive-two-steps.html#api-downloading-an-archive-two-steps-retrieval-options
// For retrieval pricing, see https://aws.amazon.com/glacier/pricing/
const RETRIEVAL_OPTION = "Bulk"

// After a Glacier restore request has been accepted, we will check
// S3 periodically to see if the item has been restored. This is the
// interval between checks.
const HOURS_BETWEEN_CHECKS = 3

// Keep the files in S3 up to 60 days, in case we're
// having system problems and we need to attempt the
// restore multiple times. We'll have other processes
// clean out the S3 bucket when necessary.
const DAYS_TO_KEEP_IN_S3 = 60

// Requests that an object be restored from Glacier to S3. This is
// the first step toward performing fixity checks on DPN bags, and
// restoring DPN bags, all of which are stored in Glacier.
type DPNGlacierRestoreInit struct {
	// Context includes logging, config, network connections, and
	// other general resources for the worker.
	Context *context.Context
	// LocalDPNRestClient lets us talk to our local DPN server.
	LocalDPNRestClient *dpn_network.DPNRestClient
	// RequestChannel is for requesting an item be moved from Glacier
	// into S3.
	RequestChannel chan *models.DPNGlacierRestoreState
	// CleanupChannel is for housekeeping, like updating NSQ.
	CleanupChannel chan *models.DPNGlacierRestoreState
	// PostTestChannel is for testing only. In production, nothing listens
	// on this channel.
	PostTestChannel chan *models.DPNGlacierRestoreState
	// S3Url is a custom URL that the S3 client should connect to.
	// We use this only in testing, when we want the client to talk
	// to a local test server. This should not be set in demo or
	// production.
	S3Url string
}

func DPNNewGlacierRestoreInit(_context *context.Context) (*DPNGlacierRestoreInit, error) {
	restorer := &DPNGlacierRestoreInit{
		Context: _context,
	}
	// Set up buffered channels
	restorerBufferSize := _context.Config.DPN.DPNGlacierRestoreWorker.NetworkConnections * 4
	workerBufferSize := _context.Config.DPN.DPNGlacierRestoreWorker.Workers * 10
	restorer.RequestChannel = make(chan *models.DPNGlacierRestoreState, restorerBufferSize)
	restorer.CleanupChannel = make(chan *models.DPNGlacierRestoreState, workerBufferSize)
	// Set up a limited number of go routines to handle the work.
	for i := 0; i < _context.Config.DPN.DPNGlacierRestoreWorker.NetworkConnections; i++ {
		go restorer.RequestRestore()
	}
	for i := 0; i < _context.Config.DPN.DPNGlacierRestoreWorker.Workers; i++ {
		go restorer.Cleanup()
	}
	// Set up a client to talk to our local DPN server.
	var err error
	restorer.LocalDPNRestClient, err = dpn_network.NewDPNRestClient(
		_context.Config.DPN.RestClient.LocalServiceURL,
		_context.Config.DPN.RestClient.LocalAPIRoot,
		_context.Config.DPN.RestClient.LocalAuthToken,
		_context.Config.DPN.LocalNode,
		_context.Config.DPN)
	return restorer, err
}

// This is the callback that NSQ workers use to handle messages from NSQ.
func (restorer *DPNGlacierRestoreInit) HandleMessage(message *nsq.Message) error {
	message.DisableAutoResponse()

	state := restorer.GetRestoreState(message)
	state.DPNWorkItem.Status = constants.StatusStarted
	restorer.SaveDPNWorkItem(state)
	if state.ErrorMessage != "" {
		restorer.Context.MessageLog.Error("Error setting up state for WorkItem %s: %s",
			string(message.Body), state.ErrorMessage)
		// No use proceeding...
		restorer.CleanupChannel <- state
		return fmt.Errorf(state.ErrorMessage)
	}
	if state.DPNWorkItem.IsCompletedOrCancelled() {
		restorer.Context.MessageLog.Info("Skipping WorkItem %d because status is %s",
			state.DPNWorkItem.Id, state.DPNWorkItem.Status)
		restorer.CleanupChannel <- state
		return nil
	}

	// OK, we're good. Ask Glacier to move the file into S3.
	restorer.RequestChannel <- state
	return nil
}

func (restorer *DPNGlacierRestoreInit) RequestRestore() {
	for state := range restorer.RequestChannel {
		requestNeeded, err := restorer.RestoreRequestNeeded(state)
		if err != nil {
			state.ErrorMessage = fmt.Sprintf("Error processing S3 HEAD request for %s: %v", state.GlacierKey, err)
		} else if requestNeeded {
			restorer.InitializeRetrieval(state)
		}
		restorer.CleanupChannel <- state
	}
}

func (restorer *DPNGlacierRestoreInit) Cleanup() {
	for state := range restorer.CleanupChannel {
		if state.ErrorMessage != "" {
			restorer.FinishWithError(state)
		} else {
			restorer.FinishWithSuccess(state)
		}
		// For testing only. The test code creates the PostTestChannel.
		// When running in demo & production, this channel is nil.
		if restorer.PostTestChannel != nil {
			restorer.PostTestChannel <- state
		}
	}
}

func (restorer *DPNGlacierRestoreInit) FinishWithSuccess(state *models.DPNGlacierRestoreState) {
	state.DPNWorkItem.ClearNodeAndPid()
	note := fmt.Sprintf("Glacier restore initiated. Will check availability "+
		"in S3 every %d hours.", HOURS_BETWEEN_CHECKS)
	if state.IsAvailableInS3 {
		note = "Item is available in S3 for download."
		state.DPNWorkItem.Note = &note
		state.DPNWorkItem.Stage = constants.StageAvailableInS3
		restorer.SaveDPNWorkItem(state)
		restorer.SendToDownloadQueue(state)
	} else {
		state.DPNWorkItem.Note = &note
		restorer.Context.MessageLog.Info("Requested %s from Glacier. %s", state.GlacierKey, note)
		state.DPNWorkItem.Retry = true
		restorer.SaveDPNWorkItem(state)
		state.NSQMessage.Requeue(HOURS_BETWEEN_CHECKS * time.Hour)
	}
}

func (restorer *DPNGlacierRestoreInit) SendToDownloadQueue(state *models.DPNGlacierRestoreState) {
	state.NSQMessage.Finish()
	topic := restorer.Context.Config.DPN.DPNS3DownloadWorker.NsqTopic
	err := restorer.Context.NSQClient.Enqueue(topic, state.DPNWorkItem.Id)
	if err != nil {
		state.ErrorMessage = fmt.Sprintf("Glacier requested succeeded, but error pushing "+
			"DPNWorkItem %d (%s) into NSQ topic %s: %v",
			state.DPNWorkItem.Id, state.DPNWorkItem.Identifier, topic, err)
		restorer.Context.MessageLog.Error(state.ErrorMessage)
		restorer.SaveDPNWorkItem(state)
	}
}

func (restorer *DPNGlacierRestoreInit) FinishWithError(state *models.DPNGlacierRestoreState) {
	state.DPNWorkItem.ClearNodeAndPid()
	state.DPNWorkItem.Note = &state.ErrorMessage
	restorer.Context.MessageLog.Error(state.ErrorMessage)

	attempts := int(state.NSQMessage.Attempts)
	maxAttempts := int(restorer.Context.Config.DPN.DPNGlacierRestoreWorker.MaxAttempts)

	if state.ErrorIsFatal {
		restorer.Context.MessageLog.Error("Error for %s is fatal. Not requeueing.", state.GlacierKey)
		state.DPNWorkItem.Status = constants.StatusFailed
		state.DPNWorkItem.Retry = false
		state.NSQMessage.Finish()
	} else if attempts > maxAttempts {
		restorer.Context.MessageLog.Error("Attempt to restore %s failed %d times. Not requeuing.",
			attempts, state.GlacierKey)
		state.DPNWorkItem.Status = constants.StatusFailed
		state.DPNWorkItem.Retry = false
		state.NSQMessage.Finish()
	} else {
		restorer.Context.MessageLog.Info("Error for %s is transient. Requeueing.", state.GlacierKey)
		state.DPNWorkItem.Retry = true
		state.NSQMessage.Requeue(1 * time.Minute)
	}

	restorer.SaveDPNWorkItem(state)
}

func (restorer *DPNGlacierRestoreInit) RestoreRequestNeeded(state *models.DPNGlacierRestoreState) (bool, error) {
	needsRestoreRequest := false
	s3Client := apt_network.NewS3Head(
		restorer.Context.Config.GetAWSAccessKeyId(),
		restorer.Context.Config.GetAWSSecretAccessKey(),
		restorer.Context.Config.DPN.DPNGlacierRegion,
		state.GlacierBucket)
	// Hack for testing: Tell the client to talk to our own
	// local S3 test server, and clear the bucket name,
	// because that gets prepended to the URL.
	if restorer.S3Url != "" {
		restorer.Context.MessageLog.Warning("Setting S3 URL to %s. This should happen only in testing!",
			restorer.S3Url)
		s3Client.SetSessionEndpoint(restorer.S3Url)
		s3Client.BucketName = ""
	}

	// Ask S3 about the status of this object
	s3Client.Head(state.GlacierKey)

	// Status 409: Conflict is an expected response.
	// It means a restore request has already been initiated.
	if strings.Contains(s3Client.ErrorMessage, "Conflict") {
		restorer.Context.MessageLog.Info("Already in progress: %s ", state.GlacierKey)
		state.RequestAccepted = true
		state.RequestedAt = time.Now().UTC()
		return false, nil
	}

	restoreRequestInfo, err := s3Client.GetRestoreRequestInfo()
	if restoreRequestInfo.RequestInProgress {
		// Log and go on
		restorer.Context.MessageLog.Info("Already in progress: %s ", state.GlacierKey)
		state.RequestAccepted = true
		state.RequestedAt = time.Now().UTC()
	} else if restoreRequestInfo.RequestIsComplete {
		// Log and update expiry date
		state.RequestAccepted = true
		state.RequestedAt = time.Now().UTC()
		state.IsAvailableInS3 = true
		state.EstimatedDeletionFromS3 = restoreRequestInfo.S3ExpiryDate
		restorer.Context.MessageLog.Info("Already restored to S3: %s", state.GlacierKey)
	} else {
		// Not restored yet and not even requested.
		// We need to make a request for this now.
		restorer.Context.MessageLog.Info("Needs Glacier retrieval request: %s", state.GlacierKey)
		needsRestoreRequest = true
	}
	return needsRestoreRequest, err
}

func (restorer *DPNGlacierRestoreInit) InitializeRetrieval(state *models.DPNGlacierRestoreState) {
	// Request restore from Glacier
	restorer.Context.MessageLog.Info("Requesting Glacier retrieval of %s from %s",
		state.GlacierKey, state.GlacierBucket)

	restoreClient := apt_network.NewS3Restore(
		restorer.Context.Config.GetAWSAccessKeyId(),
		restorer.Context.Config.GetAWSSecretAccessKey(),
		restorer.Context.Config.DPN.DPNGlacierRegion,
		state.GlacierBucket,
		state.GlacierKey,
		RETRIEVAL_OPTION,
		DAYS_TO_KEEP_IN_S3)

	// Custom S3Url is for testing only.
	if restorer.S3Url != "" {
		restorer.Context.MessageLog.Warning("Setting S3 URL to %s. This should happen only in testing!",
			restorer.S3Url)
		restoreClient.TestURL = restorer.S3Url
		restoreClient.BucketName = ""
	}

	// Figure out approximately how long this item will
	// be available in S3, once we restore it.
	now := time.Now().UTC()
	estimatedDeletionFromS3 := now.AddDate(0, 0, DAYS_TO_KEEP_IN_S3)

	// This is where me make the actual request to Glacier.
	restoreClient.Restore()
	if restoreClient.ErrorMessage != "" {
		state.ErrorMessage = fmt.Sprintf("Glacier retrieval request returned an error for %s at %s: %v",
			state.GlacierBucket, state.GlacierKey, restoreClient.ErrorMessage)
		restorer.Context.MessageLog.Error("Bad response from Glacier. Requested %s/%s. Got:\n %v",
			state.GlacierBucket, state.GlacierKey, restoreClient.Response)
	}

	// Update this info.
	state.RequestAccepted = restoreClient.RequestAccepted()
	state.RequestedAt = now
	state.EstimatedDeletionFromS3 = estimatedDeletionFromS3
	state.IsAvailableInS3 = restoreClient.AlreadyInActiveTier

	if restoreClient.RequestRejectedServiceUnavailable {
		state.ErrorMessage = fmt.Sprintf("Request to restore %s/%s: "+
			"Glacier restore service is temporarily unavailable. Try again later.",
			state.GlacierBucket, state.GlacierKey)
		state.ErrorIsFatal = false
	}
}

// GetWorkItem returns the WorkItem with the specified Id from Pharos,
// or nil.
func (restorer *DPNGlacierRestoreInit) GetRestoreState(message *nsq.Message) *models.DPNGlacierRestoreState {
	msgBody := strings.TrimSpace(string(message.Body))
	restorer.Context.MessageLog.Info("NSQ Message body: '%s'", msgBody)
	state := &models.DPNGlacierRestoreState{
		NSQMessage: message,
	}

	// Get the DPN work item
	dpnWorkItemId, err := strconv.Atoi(string(msgBody))
	if err != nil || dpnWorkItemId == 0 {
		state.ErrorMessage = fmt.Sprintf("Could not get DPNWorkItem Id from NSQ message body: %v", err)
		return state
	}
	resp := restorer.Context.PharosClient.DPNWorkItemGet(dpnWorkItemId)
	if resp.Error != nil {
		state.ErrorMessage = fmt.Sprintf("Error getting DPNWorkItem %d from Pharos: %v", dpnWorkItemId, resp.Error)
		return state
	}
	dpnWorkItem := resp.DPNWorkItem()
	if dpnWorkItem == nil {
		state.ErrorMessage = fmt.Sprintf("Pharos returned nil for WorkItem %d", dpnWorkItemId)
		return state
	}
	state.DPNWorkItem = dpnWorkItem
	state.DPNWorkItem.SetNodeAndPid()
	note := "Requesting Glacier restoration for fixity"
	state.DPNWorkItem.Note = &note

	// Get the DPN Bag from the DPN REST server.
	dpnResp := restorer.LocalDPNRestClient.DPNBagGet(dpnWorkItem.Identifier)
	if dpnResp.Error != nil {
		state.ErrorMessage = fmt.Sprintf("Error getting DPN bag %s from %s: %v", dpnWorkItem.Identifier,
			restorer.Context.Config.DPN.RestClient.LocalServiceURL, resp.Error)
		return state
	}
	dpnBag := dpnResp.Bag()
	if dpnBag == nil {
		state.ErrorMessage = fmt.Sprintf("DPN REST server returned nil for bag %s", dpnWorkItem.Identifier)
		return state
	}
	state.DPNBag = dpnBag

	// Although this is duplicate info, we record it in the state object
	// so we can see it in the Pharos UI when we're checking on the state
	// of an item.
	state.GlacierBucket = restorer.Context.Config.DPN.DPNPreservationBucket
	state.GlacierKey = dpnBag.UUID

	return state
}

func (restorer *DPNGlacierRestoreInit) SaveDPNWorkItem(state *models.DPNGlacierRestoreState) {
	jsonData, err := state.ToJson()
	if err != nil {
		msg := fmt.Sprintf("Could not marshal DPNGlacierRestoreState "+
			"for DPNWorkItem %d: %v", state.DPNWorkItem.Id, err)
		restorer.Context.MessageLog.Error(msg)
		note := "[JSON serialization error]"
		state.DPNWorkItem.Note = &note
	}

	// Update the DPNWorkItem
	state.DPNWorkItem.State = &jsonData
	state.DPNWorkItem.Retry = !state.ErrorIsFatal

	resp := restorer.Context.PharosClient.DPNWorkItemSave(state.DPNWorkItem)
	if resp.Error != nil {
		msg := fmt.Sprintf("Could not save DPNWorkItem %d "+
			"for fixity on bag %s to Pharos: %v",
			state.DPNWorkItem.Id, state.DPNWorkItem.Identifier, err)
		restorer.Context.MessageLog.Error(msg)
		if state.ErrorMessage == "" {
			state.ErrorMessage = msg
		}
	}
}
