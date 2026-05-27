package main

import (
	"fmt"
	"os"
	"path/filepath"
)

var googleServices = []string{"gmail", "drive", "calendar", "chat", "people"}

func cmdPrereq(args []string) int {
	var targetDir string
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dir":
			i++
			if i < len(args) {
				targetDir = args[i]
			}
		default:
			fmt.Fprintf(os.Stderr, "unknown flag %q\n", args[i])
			return 2
		}
	}
	if err := ensurePrereqs(targetDir); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func ensurePrereqs(targetDir string) error {
	dir, err := resolveInstallDir(targetDir)
	if err != nil {
		return err
	}
	helperPath := filepath.Join(dir, "google-workspace-mcp-auth")
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return fmt.Errorf("create %s: %w", dir, err)
	}
	if err := installGoogleHelperBinary(dir); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "✓ installed %s\n", helperPath)
	if err := installSlackHelperBinary(dir); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "✓ installed %s\n", filepath.Join(dir, "slack-mcp-auth"))
	if err := installHubSpotHelperBinary(dir); err != nil {
		return err
	}
	fmt.Fprintf(os.Stderr, "✓ installed %s\n", filepath.Join(dir, "hubspot-mcp-auth"))
	for _, s := range googleServices {
		wrapper := filepath.Join(dir, "google-workspace-mcp-auth-"+s)
		body := fmt.Sprintf("#!/bin/sh\nexec \"%s\" %s \"$@\"\n", helperPath, s)
		if err := os.WriteFile(wrapper, []byte(body), 0o755); err != nil {
			return fmt.Errorf("write wrapper %s: %w (try: sudo ally3p prereq)", wrapper, err)
		}
	}
	fmt.Fprintf(os.Stderr, "✓ installed wrappers in %s\n", dir)
	return nil
}

func needsPrereqs(helperDir string) bool {
	dir, err := resolveInstallDir(helperDir)
	if err != nil {
		return true
	}
	for _, s := range googleServices {
		if _, err := os.Stat(filepath.Join(dir, "google-workspace-mcp-auth-"+s)); err == nil {
			return false
		}
	}
	return true
}

func resolveInstallDir(override string) (string, error) {
	if override != "" {
		return override, nil
	}
	return "/usr/local/bin", nil
}
