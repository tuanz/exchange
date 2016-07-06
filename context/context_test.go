package context_test

import (
	"github.com/APTrust/exchange/context"
	"github.com/APTrust/exchange/models"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"os"
	"path"
	"path/filepath"
	"testing"
)

func TestNewContext(t *testing.T) {
	configFile := filepath.Join("config", "aptrust_test.json")
	appConfig, err := models.LoadConfigFile(configFile)
	require.Nil(t, err)

	// In some tests we want to log to STDERR, but in this case, if it
	// happens to be turned on, it just creates useless, annoying output.
	appConfig.LogToStderr = false

	_context := context.NewContext(appConfig)
	require.NotNil(t, _context)

	expectedPathToLogFile := filepath.Join(_context.Config.AbsLogDirectory(), path.Base(os.Args[0])+".log")

	assert.NotNil(t, _context.Config)
	assert.NotNil(t, _context.PharosClient)
	assert.NotNil(t, _context.MessageLog)
	assert.Equal(t, expectedPathToLogFile, _context.PathToLogFile())
	assert.Equal(t, int64(0), _context.Succeeded())
	assert.Equal(t, int64(0), _context.Failed())

	assert.NotPanics(t, func() { _context.MessageLog.Info("Test INFO log message") })
	assert.NotPanics(t, func() { _context.MessageLog.Debug("Test DEBUG log message") })

	// Cleanup, but only if context was successfully created
	if _context != nil && _context.PathToLogFile() != "" {
		os.Remove(_context.PathToLogFile())
	}
}
