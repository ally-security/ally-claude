package install

import (
	"archive/zip"
	"bytes"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"

	"github.com/anthropics/claude-3p-helper/internal/policy"
)

func installPlugin(baseDir string, b policy.Bundle) error {
	data, err := fetch(b.Source)
	if err != nil {
		return fmt.Errorf("fetch %s: %w", b.Source, err)
	}
	if err := verifySHA256(data, b.SHA256); err != nil {
		return err
	}
	dest := filepath.Join(baseDir, b.Name)
	stamp := filepath.Join(dest, ".synced-sha256")
	if existing, err := os.ReadFile(stamp); err == nil && b.SHA256 != "" {
		if strings.EqualFold(strings.TrimSpace(string(existing)), b.SHA256) {
			return nil // already up to date
		}
	}

	// Stage into a sibling tmp dir, then swap.
	tmp := dest + ".tmp"
	_ = os.RemoveAll(tmp)
	if err := os.MkdirAll(tmp, 0o755); err != nil {
		return err
	}
	if err := unzipBytes(data, tmp); err != nil {
		_ = os.RemoveAll(tmp)
		return fmt.Errorf("unzip plugin %s: %w", b.Name, err)
	}
	sum := b.SHA256
	if sum == "" {
		sum = sha256Hex(data)
	}
	if err := os.WriteFile(filepath.Join(tmp, ".synced-sha256"), []byte(sum), 0o644); err != nil {
		return err
	}

	_ = os.RemoveAll(dest)
	if err := os.Rename(tmp, dest); err != nil {
		return fmt.Errorf("swap %s: %w", dest, err)
	}
	return nil
}

func unzipBytes(data []byte, dest string) error {
	zr, err := zip.NewReader(bytes.NewReader(data), int64(len(data)))
	if err != nil {
		return err
	}
	for _, f := range zr.File {
		// Zip slip guard.
		target := filepath.Join(dest, f.Name)
		rel, err := filepath.Rel(dest, target)
		if err != nil || strings.HasPrefix(rel, "..") {
			return fmt.Errorf("unsafe path in zip: %s", f.Name)
		}
		if f.FileInfo().IsDir() {
			if err := os.MkdirAll(target, f.Mode()); err != nil {
				return err
			}
			continue
		}
		if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
			return err
		}
		rc, err := f.Open()
		if err != nil {
			return err
		}
		out, err := os.OpenFile(target, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, f.Mode())
		if err != nil {
			rc.Close()
			return err
		}
		if _, err := io.Copy(out, rc); err != nil {
			rc.Close()
			out.Close()
			return err
		}
		rc.Close()
		out.Close()
	}
	return nil
}
