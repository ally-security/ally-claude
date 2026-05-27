//go:build !darwin

package googleworkspace

import "errors"

func keychainGet(service, account string) ([]byte, error) {
	return nil, errKeychainNotFound
}

func keychainSet(service, account string, data []byte) error {
	return errors.New("client credential keychain is only supported on macOS; use --file on this platform")
}

func keychainDelete(service, account string) error {
	return nil
}
