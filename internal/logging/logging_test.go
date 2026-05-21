package logging

import (
	"log/slog"
	"testing"
)

func TestSetupLevels(t *testing.T) {
	Setup(false)
	if !slog.Default().Enabled(nil, slog.LevelInfo) {
		t.Error("INFO should be enabled at default level")
	}
	if slog.Default().Enabled(nil, slog.LevelDebug) {
		t.Error("DEBUG should be disabled at default level")
	}

	Setup(true)
	if !slog.Default().Enabled(nil, slog.LevelDebug) {
		t.Error("DEBUG should be enabled when verbose=true")
	}
}
