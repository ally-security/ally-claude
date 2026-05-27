package googleworkspace

import (
	"errors"
	"strings"
)

var errKeychainNotFound = errors.New("keychain entry not found")

func errorsIsNotFound(err error) bool {
	return err == errKeychainNotFound ||
		strings.Contains(err.Error(), "could not be found")
}
