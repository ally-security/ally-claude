//go:build !darwin

package slackmcp

import "fmt"

func keychainGet(service, account string) ([]byte, error) {
	return nil, fmt.Errorf("keychain not supported on this platform")
}

func keychainSet(service, account string, data []byte) error {
	return fmt.Errorf("keychain not supported on this platform")
}
