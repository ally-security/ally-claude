//go:build darwin

package hubspotmcp

import (
	"fmt"
	"os/exec"
	"strings"
)

func keychainGet(service, account string) ([]byte, error) {
	out, err := exec.Command(
		"security", "find-generic-password",
		"-s", service,
		"-a", account,
		"-w",
	).CombinedOutput()
	text := strings.TrimSpace(string(out))
	if err != nil {
		lower := strings.ToLower(text + " " + err.Error())
		if strings.Contains(lower, "could not be found") {
			return nil, errKeychainNotFound
		}
		return nil, fmt.Errorf("keychain read: %w: %s", err, text)
	}
	return []byte(text), nil
}

func keychainSet(service, account string, data []byte) error {
	cmd := exec.Command(
		"security", "add-generic-password",
		"-U",
		"-a", account,
		"-s", service,
		"-w", string(data),
	)
	if out, err := cmd.CombinedOutput(); err != nil {
		return fmt.Errorf("keychain write: %w: %s", err, strings.TrimSpace(string(out)))
	}
	return nil
}
