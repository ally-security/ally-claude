package slackmcp

import (
	"encoding/json"
	"fmt"
	"strings"
)

const KeychainService = "slack-mcp-auth"

const (
	ClientKeychainAccount = "oauth-client"
	UserTokenAccount        = "user-token"
)

type ClientCredentials struct {
	ClientID     string `json:"clientId"`
	ClientSecret string `json:"clientSecret"`
	CallbackPort int    `json:"callbackPort"`
}

func SaveClientCredentials(clientID, clientSecret string, callbackPort int) error {
	clientID = strings.TrimSpace(clientID)
	clientSecret = strings.TrimSpace(clientSecret)
	if clientID == "" || clientSecret == "" {
		return fmt.Errorf("slack client_id and client_secret are required")
	}
	if callbackPort == 0 {
		callbackPort = DefaultCallbackPort
	}
	data, err := json.Marshal(ClientCredentials{
		ClientID:     clientID,
		ClientSecret: clientSecret,
		CallbackPort: callbackPort,
	})
	if err != nil {
		return err
	}
	return keychainSet(KeychainService, ClientKeychainAccount, data)
}

func LoadClientCredentials() (ClientCredentials, error) {
	data, err := keychainGet(KeychainService, ClientKeychainAccount)
	if err != nil {
		return ClientCredentials{}, fmt.Errorf("no Slack OAuth client in Keychain — run: ally3p claude sync <policy.yaml>")
	}
	var c ClientCredentials
	if err := json.Unmarshal(data, &c); err != nil {
		return ClientCredentials{}, err
	}
	if c.ClientID == "" || c.ClientSecret == "" {
		return ClientCredentials{}, fmt.Errorf("Slack OAuth client in Keychain is incomplete — re-run ally3p claude sync")
	}
	if c.CallbackPort == 0 {
		c.CallbackPort = DefaultCallbackPort
	}
	return c, nil
}

func HasClientCredentials() bool {
	_, err := LoadClientCredentials()
	return err == nil
}
