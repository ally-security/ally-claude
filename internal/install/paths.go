package install

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
)

type Paths struct {
	ConfigLibrary string // <id>.json + _meta.json live here
	OrgPlugins    string // unzipped plugin bundles
	Extensions    string // .mcpb files
}

func resolvePaths() (Paths, error) {
	switch runtime.GOOS {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, err
		}
		base := filepath.Join(home, "Library", "Application Support", "Claude-3p")
		return Paths{
			ConfigLibrary: filepath.Join(base, "configLibrary"),
			OrgPlugins:    filepath.Join(base, "orgPlugins"),
			Extensions:    filepath.Join(base, "extensions"),
		}, nil
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			return Paths{}, fmt.Errorf("LOCALAPPDATA not set")
		}
		base = filepath.Join(base, "Claude-3p")
		return Paths{
			ConfigLibrary: filepath.Join(base, "configLibrary"),
			OrgPlugins:    filepath.Join(base, "orgPlugins"),
			Extensions:    filepath.Join(base, "extensions"),
		}, nil
	default:
		return Paths{}, fmt.Errorf("unsupported OS: %s (only darwin and windows are supported)", runtime.GOOS)
	}
}
