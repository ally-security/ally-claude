package main

import (
	"embed"
	"fmt"
	"os"
	"path/filepath"
)

// helperFS holds the embedded google-workspace-mcp-auth binary.
// The file must exist at build time (produced by `make build-bin`).
//
//go:embed embedded/google-workspace-mcp-auth embedded/slack-mcp-auth embedded/hubspot-mcp-auth
var helperFS embed.FS

func installGoogleHelperBinary(targetDir string) error {
	return installEmbeddedBinary("embedded/google-workspace-mcp-auth", targetDir, "google-workspace-mcp-auth")
}

func installSlackHelperBinary(targetDir string) error {
	return installEmbeddedBinary("embedded/slack-mcp-auth", targetDir, "slack-mcp-auth")
}

func installHubSpotHelperBinary(targetDir string) error {
	return installEmbeddedBinary("embedded/hubspot-mcp-auth", targetDir, "hubspot-mcp-auth")
}

func installEmbeddedBinary(embedPath, targetDir, name string) error {
	data, err := helperFS.ReadFile(embedPath)
	if err != nil {
		return fmt.Errorf("embedded %s not found (run make build): %w", name, err)
	}
	dst := filepath.Join(targetDir, name)
	if err := os.WriteFile(dst, data, 0o755); err != nil {
		return fmt.Errorf("write %s: %w (try: sudo ally3p prereq)", dst, err)
	}
	return nil
}
