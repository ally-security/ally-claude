package googleworkspace

import (
	"encoding/json"
	"errors"
	"fmt"
)

const (
	// ClientKeychainAccount stores the enterprise OAuth app (one per machine).
	ClientKeychainAccount = "oauth-client"
)

// SaveClientCredentials stores OAuth client id/secret in macOS Keychain (once per machine).
func SaveClientCredentials(clientID, clientSecret string) error {
	if clientID == "" || clientSecret == "" {
		return errors.New("client id and secret are required")
	}
	payload, err := json.Marshal(credentialsFile{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		return err
	}
	if err := keychainSet(KeychainServiceNameForStore(), ClientKeychainAccount, payload); err != nil {
		return err
	}
	return nil
}

// SaveClientCredentialsForService stores OAuth client id/secret in Keychain scoped to a
// single Workspace service (e.g. gmail). This allows different client secrets per service.
func SaveClientCredentialsForService(serviceID, clientID, clientSecret string) error {
	if serviceID == "" {
		return errors.New("service id is required")
	}
	if clientID == "" || clientSecret == "" {
		return errors.New("client id and secret are required")
	}
	payload, err := json.Marshal(credentialsFile{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	})
	if err != nil {
		return err
	}
	return keychainSet(KeychainServiceNameForStore(), clientKeychainAccountForService(serviceID), payload)
}

// DeleteClientCredentials removes the enterprise OAuth client from Keychain.
func DeleteClientCredentials() error {
	return keychainDelete(KeychainServiceNameForStore(), ClientKeychainAccount)
}

// DeleteClientCredentialsForService removes the enterprise OAuth client secret for a specific service.
func DeleteClientCredentialsForService(serviceID string) error {
	if serviceID == "" {
		return errors.New("service id is required")
	}
	return keychainDelete(KeychainServiceNameForStore(), clientKeychainAccountForService(serviceID))
}

// ClientCredentialsKeychainLocation describes where IT-provisioned secrets live.
func ClientCredentialsKeychainLocation() string {
	return fmt.Sprintf("Keychain service %q account %q", KeychainServiceNameForStore(), ClientKeychainAccount)
}

func credentialsFromKeychain(svc Service) (ClientCredentials, error) {
	accounts := []string{clientKeychainAccountForService(svc.ID), ClientKeychainAccount}
	for _, account := range accounts {
		data, err := keychainGet(KeychainServiceNameForStore(), account)
		if err != nil {
			if errorsIsNotFound(err) {
				continue
			}
			return ClientCredentials{}, err
		}
		var f credentialsFile
		if err := json.Unmarshal(data, &f); err != nil {
			return ClientCredentials{}, fmt.Errorf("keychain %s: invalid credentials JSON: %w", account, err)
		}
		if f.ClientID == "" || f.ClientSecret == "" {
			continue
		}
		return ClientCredentials{
			ClientID:     f.ClientID,
			ClientSecret: f.ClientSecret,
			CallbackPort: f.CallbackPort,
			Scope:        f.Scope,
			Source:       ClientCredentialsKeychainLocation(),
		}, nil
	}
	return ClientCredentials{}, errKeychainNotFound
}

func clientKeychainAccountForService(serviceID string) string {
	return "oauth-client-" + serviceID
}

func mergeClientCredentials(base, overlay ClientCredentials) ClientCredentials {
	out := base
	if overlay.ClientID != "" {
		out.ClientID = overlay.ClientID
	}
	if overlay.ClientSecret != "" {
		out.ClientSecret = overlay.ClientSecret
	}
	if overlay.CallbackPort != 0 {
		out.CallbackPort = overlay.CallbackPort
	}
	if overlay.Scope != "" {
		out.Scope = overlay.Scope
	}
	if overlay.Source != "" && out.Source == "" {
		out.Source = overlay.Source
	}
	return out
}
