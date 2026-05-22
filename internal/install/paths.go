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

// pathResolver is the indirection tests use to inject paths.
var pathResolver = func() (Paths, error) {
	return ResolvePathsFor(runtime.GOOS)
}

func resolvePaths() (Paths, error) { return pathResolver() }

// SetPathResolverForTest overrides path resolution for tests outside this
// package. The returned restore func should be deferred.
func SetPathResolverForTest(fn func() (Paths, error)) (restore func()) {
	prev := pathResolver
	pathResolver = fn
	return func() { pathResolver = prev }
}

// ResolvePathsFor computes the install layout for a given GOOS using the
// caller's $HOME (and %LOCALAPPDATA% on windows). Exposed for tests.
func ResolvePathsFor(goos string) (Paths, error) {
	switch goos {
	case "darwin":
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, err
		}
		base := filepath.Join(home, "Library", "Application Support", "Claude-3p")
		return layout(base), nil
	case "windows":
		base := os.Getenv("LOCALAPPDATA")
		if base == "" {
			return Paths{}, fmt.Errorf("LOCALAPPDATA not set")
		}
		return layout(filepath.Join(base, "Claude-3p")), nil
	case "linux":
		home, err := os.UserHomeDir()
		if err != nil {
			return Paths{}, err
		}
		return layout(filepath.Join(home, ".config", "Claude-3p")), nil
	default:
		return Paths{}, fmt.Errorf("unsupported OS: %s", goos)
	}
}

func layout(base string) Paths {
	return Paths{
		ConfigLibrary: filepath.Join(base, "configLibrary"),
		OrgPlugins:    filepath.Join(base, "orgPlugins"),
		Extensions:    filepath.Join(base, "extensions"),
	}
}
