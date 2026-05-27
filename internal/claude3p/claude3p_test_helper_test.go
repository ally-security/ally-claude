package claude3p_test

import (
	"os"
	"path/filepath"
	"testing"
)

// setTestClaude3PHome points CLAUDE_3P_HOME at a temp configLibrary so Sync works in CI.
func setTestClaude3PHome(t *testing.T) {
	t.Helper()
	dir := t.TempDir()
	lib := filepath.Join(dir, "configLibrary")
	if err := os.MkdirAll(lib, 0o700); err != nil {
		t.Fatal(err)
	}
	const id = "00000000-0000-4000-8000-000000000001"
	if err := os.WriteFile(filepath.Join(lib, "_meta.json"), []byte(`{"activeConfigId":"`+id+`"}`), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_3P_HOME", dir)
}
