//go:build !darwin

package googleworkspace

import (
	"fmt"
	"os"
	"path/filepath"
)

func tokenFilePath(account string) string {
	if p := os.Getenv("GOOGLE_MCP_TOKEN_FILE"); p != "" {
		return p
	}
	dir, err := os.UserConfigDir()
	if err != nil {
		dir = "."
	}
	return filepath.Join(dir, "claude-3p", account+".json")
}

func (s Store) tokenAccounts() []string {
	if s.Account == SharedUserKeychainAccount {
		return []string{SharedUserKeychainAccount}
	}
	return []string{SharedUserKeychainAccount, s.Account}
}

func (s Store) Load() (*Token, error) {
	for _, account := range s.tokenAccounts() {
		data, err := os.ReadFile(tokenFilePath(account))
		if err != nil {
			if os.IsNotExist(err) {
				continue
			}
			return nil, err
		}
		return ParseToken(data)
	}
	return nil, fmt.Errorf("no Google token — run: ally3p claude login")
}

func (s Store) Save(t *Token) error {
	path := tokenFilePath(SharedUserKeychainAccount)
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	data, err := t.Marshal()
	if err != nil {
		return err
	}
	return os.WriteFile(path, data, 0o600)
}

func (s Store) Delete() error {
	for _, account := range s.tokenAccounts() {
		err := os.Remove(tokenFilePath(account))
		if err != nil && !os.IsNotExist(err) {
			return err
		}
	}
	return nil
}
