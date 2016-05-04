package models_test

import (
	"github.com/APTrust/exchange/constants"
	"github.com/APTrust/exchange/models"
	"github.com/stretchr/testify/assert"
	"testing"
)

func TestEventTypeValid(t *testing.T) {
	for _, eventType := range constants.EventTypes {
		premisEvent := &models.PremisEvent{
			EventType: eventType,
		}
		if premisEvent.EventTypeValid() == false {
			t.Errorf("EventType '%s' should be valid", eventType)
		}
	}
	premisEvent := &models.PremisEvent{
		EventType: "pub_crawl",
	}
	if premisEvent.EventTypeValid() == true {
		t.Errorf("EventType 'pub_crawl' should not be valid")
	}
}

func TestNewEventObjectIngest(t *testing.T) {
	event, err := models.NewEventObjectIngest(300)
	if err != nil {
		t.Errorf("Error creating PremisEvent: %v", err)
		return
	}
	assert.Len(t, event.Identifier, 36)
	assert.Equal(t, "ingest", event.EventType)
	assert.False(t, event.DateTime.IsZero())
	assert.Equal(t, "Copied all files to perservation bucket", event.Detail)
	assert.Equal(t, "Success", event.Outcome)
	assert.Equal(t, "300 files copied", event.OutcomeDetail)
	assert.Equal(t, "goamz S3 client", event.Object)
	assert.Equal(t, "https://github.com/crowdmob/goamz", event.Agent)
	assert.Equal(t, "Multipart put using md5 checksum", event.OutcomeInformation)
}

func TestNewEventObjectIdentifierAssignment(t *testing.T) {
	event, err := models.NewEventObjectIdentifierAssignment("test.edu/object001")
	if err != nil {
		t.Errorf("Error creating PremisEvent: %v", err)
		return
	}
	assert.Len(t, event.Identifier, 36)
	assert.Equal(t, "identifier_assignment", event.EventType)
	assert.False(t, event.DateTime.IsZero())
	assert.Equal(t, "Assigned bag identifier", event.Detail)
	assert.Equal(t, "Success", event.Outcome)
	assert.Equal(t, "test.edu/object001", event.OutcomeDetail)
	assert.Equal(t, "APTrust exchange", event.Object)
	assert.Equal(t, "https://github.com/APTrust/exchange", event.Agent)
	assert.Equal(t, "Institution domain + tar file name", event.OutcomeInformation)
}

func TestNewEventObjectRights(t *testing.T) {
	event, err := models.NewEventObjectRights("institution")
	if err != nil {
		t.Errorf("Error creating PremisEvent: %v", err)
		return
	}
	assert.Len(t, event.Identifier, 36)
	assert.Equal(t, "access_assignment", event.EventType)
	assert.False(t, event.DateTime.IsZero())
	assert.Equal(t, "Assigned bag access rights", event.Detail)
	assert.Equal(t, "Success", event.Outcome)
	assert.Equal(t, "institution", event.OutcomeDetail)
	assert.Equal(t, "APTrust exchange", event.Object)
	assert.Equal(t, "https://github.com/APTrust/exchange", event.Agent)
	assert.Equal(t, "Set access to institution", event.OutcomeInformation)
}

func TestNewEventGenericFileIngest(t *testing.T) {
	event, err := models.NewEventGenericFileIngest(TEST_TIMESTAMP, "123456789")
	if err != nil {
		t.Errorf("Error creating PremisEvent: %v", err)
		return
	}
	assert.Len(t, event.Identifier, 36)
	assert.Equal(t, "ingest", event.EventType)
	assert.Equal(t, TEST_TIMESTAMP, event.DateTime)
	assert.Equal(t, "Completed copy to S3", event.Detail)
	assert.Equal(t, "Success", event.Outcome)
	assert.Equal(t, "md5:123456789", event.OutcomeDetail)
	assert.Equal(t, "exchange + goamz S3 client", event.Object)
	assert.Equal(t, "https://github.com/APTrust/exchange", event.Agent)
	assert.Equal(t, "Put using md5 checksum", event.OutcomeInformation)
}

func TestNewEventGenericFileFixityCheck(t *testing.T) {
	event, err := models.NewEventGenericFileFixityCheck(TEST_TIMESTAMP, constants.AlgMd5, "123456789", true)
	if err != nil {
		t.Errorf("Error creating PremisEvent: %v", err)
		return
	}
	assert.Len(t, event.Identifier, 36)
	assert.Equal(t, "fixity_check", event.EventType)
	assert.Equal(t, TEST_TIMESTAMP, event.DateTime)
	assert.Equal(t, "Fixity check against registered hash", event.Detail)
	assert.Equal(t, "Success", event.Outcome)
	assert.Equal(t, "md5:123456789", event.OutcomeDetail)
	assert.Equal(t, "Go language crypto/md5", event.Object)
	assert.Equal(t, "http://golang.org/pkg/crypto/md5/", event.Agent)
	assert.Equal(t, "Fixity matches", event.OutcomeInformation)

	event, err = models.NewEventGenericFileFixityCheck(TEST_TIMESTAMP, constants.AlgSha256, "123456789", false)
	if err != nil {
		t.Errorf("Error creating PremisEvent: %v", err)
		return
	}
	assert.Len(t, event.Identifier, 36)
	assert.Equal(t, "fixity_check", event.EventType)
	assert.Equal(t, TEST_TIMESTAMP, event.DateTime)
	assert.Equal(t, "Fixity check against registered hash", event.Detail)
	assert.Equal(t, "Failed", event.Outcome)
	assert.Equal(t, "sha256:123456789", event.OutcomeDetail)
	assert.Equal(t, "Go language crypto/sha256", event.Object)
	assert.Equal(t, "http://golang.org/pkg/crypto/sha256/", event.Agent)
	assert.Equal(t, "Fixity did not match", event.OutcomeInformation)
}

func TestNewEventGenericFileFixityGeneration(t *testing.T) {
	event, err := models.NewEventGenericFileFixityGeneration(TEST_TIMESTAMP, constants.AlgMd5, "123456789")
	if err != nil {
		t.Errorf("Error creating PremisEvent: %v", err)
		return
	}
	assert.Len(t, event.Identifier, 36)
	assert.Equal(t, "fixity_generation", event.EventType)
	assert.Equal(t, TEST_TIMESTAMP, event.DateTime)
	assert.Equal(t, "Calculated new fixity value", event.Detail)
	assert.Equal(t, "Success", event.Outcome)
	assert.Equal(t, "md5:123456789", event.OutcomeDetail)
	assert.Equal(t, "Go language crypto/md5", event.Object)
	assert.Equal(t, "http://golang.org/pkg/crypto/md5/", event.Agent)
	assert.Equal(t, "", event.OutcomeInformation)

	event, err = models.NewEventGenericFileFixityGeneration(TEST_TIMESTAMP, constants.AlgSha256, "123456789")
	if err != nil {
		t.Errorf("Error creating PremisEvent: %v", err)
		return
	}
	assert.Len(t, event.Identifier, 36)
	assert.Equal(t, "fixity_generation", event.EventType)
	assert.Equal(t, TEST_TIMESTAMP, event.DateTime)
	assert.Equal(t, "Calculated new fixity value", event.Detail)
	assert.Equal(t, "Success", event.Outcome)
	assert.Equal(t, "sha256:123456789", event.OutcomeDetail)
	assert.Equal(t, "Go language crypto/sha256", event.Object)
	assert.Equal(t, "http://golang.org/pkg/crypto/sha256/", event.Agent)
	assert.Equal(t, "", event.OutcomeInformation)
}

func TestNewEventGenericFileIdentifierAssignment(t *testing.T) {
	event, err := models.NewEventGenericFileIdentifierAssignment(TEST_TIMESTAMP, constants.IdTypeBagAndPath, "blah.edu/blah/blah.txt")
	if err != nil {
		t.Errorf("Error creating PremisEvent: %v", err)
		return
	}
	assert.Len(t, event.Identifier, 36)
	assert.Equal(t, "identifier_assignment", event.EventType)
	assert.Equal(t, TEST_TIMESTAMP, event.DateTime)
	assert.Equal(t, "Assigned new institution.bag/path identifier", event.Detail)
	assert.Equal(t, "Success", event.Outcome)
	assert.Equal(t, "blah.edu/blah/blah.txt", event.OutcomeDetail)
	assert.Equal(t, "APTrust exchange/ingest processor", event.Object)
	assert.Equal(t, "https://github.com/APTrust/exchange", event.Agent)
	assert.Equal(t, "", event.OutcomeInformation)

	event, err = models.NewEventGenericFileIdentifierAssignment(TEST_TIMESTAMP, constants.IdTypeStorageURL, "https://example.com/000-000-999")
	if err != nil {
		t.Errorf("Error creating PremisEvent: %v", err)
		return
	}
	assert.Len(t, event.Identifier, 36)
	assert.Equal(t, "identifier_assignment", event.EventType)
	assert.Equal(t, TEST_TIMESTAMP, event.DateTime)
	assert.Equal(t, "Assigned new storage URL identifier", event.Detail)
	assert.Equal(t, "Success", event.Outcome)
	assert.Equal(t, "https://example.com/000-000-999", event.OutcomeDetail)
	assert.Equal(t, "Go uuid library + goamz S3 library", event.Object)
	assert.Equal(t, "http://github.com/nu7hatch/gouuid", event.Agent)
	assert.Equal(t, "", event.OutcomeInformation)

}

func TestNewEventGenericFileReplication(t *testing.T) {
	event, err := models.NewEventGenericFileReplication(TEST_TIMESTAMP, "https://example.com/123456789")
	if err != nil {
		t.Errorf("Error creating PremisEvent: %v", err)
		return
	}
	assert.Len(t, event.Identifier, 36)
	assert.Equal(t, "replication", event.EventType)
	assert.Equal(t, TEST_TIMESTAMP, event.DateTime)
	assert.Equal(t, "Copied to replication storage and assigned replication URL identifier", event.Detail)
	assert.Equal(t, "Success", event.Outcome)
	assert.Equal(t, "https://example.com/123456789", event.OutcomeDetail)
	assert.Equal(t, "Go uuid library + goamz S3 library", event.Object)
	assert.Equal(t, "http://github.com/nu7hatch/gouuid", event.Agent)
	assert.Equal(t, "", event.OutcomeInformation)
}