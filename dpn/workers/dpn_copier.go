package workers

import (
	"fmt"
	"github.com/APTrust/exchange/context"
	"github.com/APTrust/exchange/dpn/models"
	"github.com/APTrust/exchange/dpn/network"
	apt_models "github.com/APTrust/exchange/models"
	"github.com/nsqio/go-nsq"
//	"os"
	"os/exec"
//	"path/filepath"
//	"time"
)

// dpn_copier copies tarred bags from other nodes via rsync.
// This is used when replicating content from other nodes.
// For putting together DPN bags from APTrust files, see fetcher.go.

type Copier struct {
	CopyChannel         chan *CopyManifest
	ChecksumChannel     chan *CopyManifest
	Context             *context.Context
	LocalClient         *network.DPNRestClient
	RemoteClients       map[string]*network.DPNRestClient
}

type CopyManifest struct {
	NsqMessage          *nsq.Message `json:"-"`
	DPNWorkItem         *apt_models.DPNWorkItem
	ReplicationTransfer *models.ReplicationTransfer
	DPNBag              *models.DPNBag
	WorkSummary         *apt_models.WorkSummary
	LocalPath           string
	RsyncStdout         string
	RsyncStderr         string
}

func NewCopier(_context *context.Context) (*Copier, error) {
	localClient, err := network.NewDPNRestClient(
		_context.Config.DPN.RestClient.LocalServiceURL,
		_context.Config.DPN.RestClient.LocalAPIRoot,
		_context.Config.DPN.RestClient.LocalAuthToken,
		_context.Config.DPN.LocalNode,
		_context.Config.DPN)
	if err != nil {
		return nil, fmt.Errorf("Error creating local DPN REST client: %v", err)
	}
	remoteClients, err := localClient.GetRemoteClients()
	if err != nil {
		return nil, err
	}
	copier := &Copier {
		Context: _context,
		LocalClient: localClient,
		RemoteClients: remoteClients,
	}
	workerBufferSize := _context.Config.DPN.DPNCopyWorker.Workers * 4
	copier.CopyChannel = make(chan *CopyManifest, workerBufferSize)
	copier.ChecksumChannel = make(chan *CopyManifest, workerBufferSize)
	for i := 0; i < _context.Config.DPN.DPNCopyWorker.Workers; i++ {
		go copier.doCopy()
		go copier.verifyChecksum()
	}
	return copier, nil
}

func (copier *Copier) HandleMessage(message *nsq.Message) error {
	message.DisableAutoResponse()

	// Get the DPNWorkItem, the ReplicationTransfer, and the DPNBag
	copyManifest := copier.buildCopyManifest(message)

	// Start processing.
	copier.CopyChannel <- copyManifest
	copier.Context.MessageLog.Info("Put xfer request %s (bag %s) from %s " +
		" into the copy channel", copyManifest.ReplicationTransfer.ReplicationId,
		copyManifest.ReplicationTransfer, copyManifest.ReplicationTransfer.FromNode)
	return nil
}

// Copy the file from the remote node to our local staging area.
func (copier *Copier) doCopy() {
	for copyManifest := range copier.CopyChannel {
		localPath := "?"
		rsyncCommand := GetRsyncCommand(copyManifest.ReplicationTransfer.Link,
			localPath, copier.Context.Config.DPN.UseSSHWithRsync)

		// Touch message on both sides of rsync, so NSQ doesn't time out.
		if copyManifest.NsqMessage != nil {
			copyManifest.NsqMessage.Touch()
		}
		output, err := rsyncCommand.CombinedOutput()
		copier.Context.MessageLog.Info("Rsync Output: %s", output)
		if copyManifest.NsqMessage != nil {
			copyManifest.NsqMessage.Touch()
		}
		if err != nil {
			// Something went wrong
		} else {
			// OK
		}
	}
}

// Run a checksum on the tag manifest and send that back to the
// FromNode. If the checksum is good, the FromNode will set
// the ReplicationTransfer's StoreRequested attribute to true,
// and we should store the bag. If the checksum is bad, the remote
// node will set StoreRequested to false, and we should delete
// the tar file.
func (copier *Copier) verifyChecksum() {
	//for copyManifest := range copier.ChecksumChannel {
		// 1. Calculate the sha256 digest of the tag manifest.
		// 2. Send the result the ReplicationTransfer.FromNode.
		// 3. If the updated ReplicationTransfer.StoreRequested is true,
		//    push this item into the validation queue. Otherwise,
		//    delete the bag from the local staging area.
	//}
}

func (copier *Copier) buildCopyManifest(message *nsq.Message) (*CopyManifest) {
	// 1. Get the DPNWorkItem from Pharos.
	//    Stop if it's marked complete.
	// 2. Get the ReplicationTransfer from the remote node.
	//    Stop if it's completed or cancelled.
	// 3. Get the DPNBag record from the remote node.
	//    We need to know its size.
	// 4. Build and return the CopyManifest.
	return nil
}

// Make sure we have space to copy this item from the remote node.
// We will be validating this bag in a later step without untarring it,
// so we just have to reserve enough room for the tar file.
func (copier *Copier) reserveSpaceOnVolume(copyManifest *CopyManifest) (bool) {
	okToCopy := false
	err := copier.Context.VolumeClient.Ping(500)
	if err == nil {
		path := copyManifest.LocalPath
		ok, err := copier.Context.VolumeClient.Reserve(path, uint64(copyManifest.DPNBag.Size))
		if err != nil {
			copier.Context.MessageLog.Warning("Volume service returned an error. " +
				"Will requeue ReplicationTransfer %s bag (%s) because we may not " +
				"have enough space to copy %d bytes from %s.",
				copyManifest.ReplicationTransfer.ReplicationId,
				copyManifest.ReplicationTransfer.Bag,
				copyManifest.DPNBag.Size,
				copyManifest.ReplicationTransfer.FromNode)
		} else if ok {
			// VolumeService says we have enough space for this.
			okToCopy = ok
		}
	} else {
		copier.Context.MessageLog.Warning("Volume service is not running or returned an error. " +
			"Continuing as if we have enough space to download %d bytes.",
			copyManifest.DPNBag.Size,)
		okToCopy = true
	}
	return okToCopy
}



// GetRsyncCommand returns a command object for copying from the remote
// location to the local filesystem. The copy is done via rsync over ssh,
// and the command will capture stdout and stderr. The copyFrom param
// should be a valid scp target in this format:
//
// remoteuser@remotehost:/remote/dir/bag.tar
//
// The copyTo param should be an absolute path on a locally-accessible
// file system, such as:
//
// /mnt/dpn/data/bag.tar
//
// Using this assumes a few things:
//
// 1. You have rsync installed.
// 2. You have an ssh client installed.
// 3. You have an entry in your ~/.ssh/config file specifying
//    connection and key information for the remote host.
//
// Usage:
//
// command := GetRsyncCommand("aptrust@tdr:bag.tar", "/mnt/dpn/bag.tar")
// err := command.Run()
// if err != nil {
//    ... do something ...
// }
//
// -- OR --
//
// output, err := command.CombinedOutput()
// if err != nil {
//    fmt.Println(err.Error())
//    fmt.Println(string(output))
// }
func GetRsyncCommand(copyFrom, copyTo string, useSSH bool) (*exec.Cmd) {
	//rsync -avz -e ssh remoteuser@remotehost:/remote/dir /this/dir/
	if useSSH {
		return exec.Command("rsync", "-avzW", "-e",  "ssh", copyFrom, copyTo, "--inplace")
	}
	return exec.Command("rsync", "-avzW", "--inplace", copyFrom, copyTo)
}
