package nativeui

import (
	"os"
	"testing"

	"github.com/coilyco/agent-compose/v2/internal/rostertest"
)

func TestMain(m *testing.M) {
	rostertest.Use()
	os.Exit(m.Run())
}
