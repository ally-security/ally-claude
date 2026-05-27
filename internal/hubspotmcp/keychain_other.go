//go:build !darwin

package hubspotmcp

import "fmt"

func keychainGet(service, account string) ([]byte, error) {
	return nil, fmt.Errorf("hubspot keychain storage is only supported on macOS")
}

func keychainSet(service, account string, data []byte) error {
	return fmt.Errorf("hubspot keychain storage is only supported on macOS")
}
