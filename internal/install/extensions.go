package install

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/claude-3p-helper/internal/policy"
)

func installExtension(baseDir string, b policy.Bundle) error {
	dest := filepath.Join(baseDir, b.Name+".mcpb")
	if b.SHA256 != "" {
		if existing, err := os.ReadFile(dest); err == nil {
			if strings.EqualFold(sha256Hex(existing), b.SHA256) {
				return nil // already up to date
			}
		}
	}

	data, err := fetch(b.Source)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", b.Source, err)
	}
	if err := verifySHA256(data, b.SHA256); err != nil {
		return err
	}
	if err := os.MkdirAll(baseDir, 0o755); err != nil {
		return err
	}
	tmp := dest + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	return os.Rename(tmp, dest)
}
