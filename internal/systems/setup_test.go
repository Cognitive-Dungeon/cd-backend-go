package systems

import (
	"cognitive-server/pkg/logger"
	"os"
	"testing"
)

func TestMain(m *testing.M) {
	// Initialize the global logger before running any tests
	logger.Init()

	// Exit with the result of the tests
	os.Exit(m.Run())
}
