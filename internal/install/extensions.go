package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/claude-3p-helper/internal/policy"
)

func installExtension(baseDir string, b policy.Bundle) (InstallResult, error) {
	dest := filepath.Join(baseDir, b.Name+".mcpb")
	if b.SHA256 != "" {
		if existing, err := os.ReadFile(dest); err == nil {
			if strings.EqualFold(sha256Hex(existing), b.SHA256) {
				return InstallResult{Dest: dest, SHA256: b.SHA256, Skipped: true}, nil
			}
		}
	}

	data, err := fetch(b.Source)
	if err != nil {
		return InstallResult{}, fmt.Errorf("fetch %s: %w", b.Source, err)
	}
	if err := verifySHA256(data, b.SHA256); err != nil {
		return InstallResult{}, err
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return InstallResult{}, err
	}
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return InstallResult{}, err
	}
	if err := os.Rename(tmp, dest); err != nil {
		return InstallResult{}, err
	}
	sum := b.SHA256
	if sum == "" {
		sum = sha256Hex(data)
	}
	return InstallResult{Dest: dest, Bytes: len(data), SHA256: sum}, nil
}
