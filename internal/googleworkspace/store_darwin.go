//go:build darwin

package googleworkspace

import "fmt"

func (s Store) tokenAccounts() []string {
	if s.Account == SharedUserKeychainAccount {
		return []string{SharedUserKeychainAccount}
	}
	return []string{SharedUserKeychainAccount, s.Account}
}

func (s Store) Load() (*Token, error) {
	for _, account := range s.tokenAccounts() {
		data, err := keychainGet(s.Service, account)
		if err != nil {
			if errorsIsNotFound(err) {
				continue
			}
			return nil, err
		}
		return ParseToken(data)
	}
	return nil, fmt.Errorf("no Google token in keychain — run: ally3p claude login (or google-workspace-mcp-auth-gmail login)")
}

func (s Store) Save(t *Token) error {
	data, err := t.Marshal()
	if err != nil {
		return err
	}
	return keychainSet(s.Service, SharedUserKeychainAccount, data)
}

func (s Store) Delete() error {
	for _, account := range s.tokenAccounts() {
		if err := keychainDelete(s.Service, account); err != nil {
			return err
		}
	}
	return nil
}
