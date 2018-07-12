package integration_test

import (
	"github.com/APTrust/exchange/context"
	"github.com/APTrust/exchange/models"
	"github.com/APTrust/exchange/util/testutil"
	"github.com/stretchr/testify/require"
	"path/filepath"
	"strings"
	"testing"
)

// apt_mark_for_restore_test is used by scripts/integration_test.rb
// to mark some APTrust bags for restoration, so that the apt_restore
// integration test will have some bags to work with.
func TestMarkForRestore(t *testing.T) {
	if !testutil.ShouldRunIntegrationTests() {
		t.Skip("Skipping integration test. Set ENV var RUN_EXCHANGE_INTEGRATION=true if you want to run them.")
	}
	configFile := filepath.Join("config", "integration.json")
	config, err := models.LoadConfigFile(configFile)
	require.Nil(t, err)
	_context := context.NewContext(config)

	// Request a few objects for restore.
	for _, s3Key := range testutil.INTEGRATION_GOOD_BAGS[0:8] {
		identifier := strings.Replace(s3Key, "aptrust.integration.test", "test.edu", 1)
		identifier = strings.Replace(identifier, ".tar", "", 1)
		resp := _context.PharosClient.IntellectualObjectRequestRestore(identifier)
		workItem := resp.WorkItem()
		require.Nil(t, resp.Error)
		require.NotNil(t, workItem)
		_context.MessageLog.Info("Created restore request WorkItem #%d for object %s",
			workItem.Id, workItem.ObjectIdentifier)
	}

	// And request a few files too.
	files := []string{
		"test.edu/example.edu.tagsample_good/data/datastream-DC",
		"test.edu/example.edu.tagsample_good/data/datastream-MARC",
	}
	for _, gfIdentifier := range files {
		resp := _context.PharosClient.GenericFileRequestRestore(gfIdentifier)
		workItem := resp.WorkItem()
		require.Nil(t, resp.Error)
		require.NotNil(t, workItem)
		_context.MessageLog.Info("Created restore request WorkItem #%d for file %s",
			workItem.Id, gfIdentifier)
	}
}
