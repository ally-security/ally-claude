package hubspotmcp

import (
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

type UserToken struct {
	AccessToken  string    `json:"accessToken"`
	RefreshToken string    `json:"refreshToken"`
	ExpiresAt    time.Time `json:"expiresAt"`
}

func SaveUserToken(token UserToken) error {
	if strings.TrimSpace(token.AccessToken) == "" {
		return fmt.Errorf("empty HubSpot access token")
	}
	data, err := json.Marshal(token)
	if err != nil {
		return err
	}
	return keychainSet(KeychainService, UserTokenAccount, data)
}

func LoadUserToken() (UserToken, error) {
	data, err := keychainGet(KeychainService, UserTokenAccount)
	if err != nil {
		return UserToken{}, fmt.Errorf("no HubSpot user token — run: ally3p claude login [policy.yaml]")
	}
	var token UserToken
	if err := json.Unmarshal(data, &token); err != nil {
		return UserToken{}, err
	}
	if strings.TrimSpace(token.AccessToken) == "" {
		return UserToken{}, fmt.Errorf("empty HubSpot token in Keychain")
	}
	return token, nil
}

func HasUserToken() bool {
	_, err := LoadUserToken()
	return err == nil
}
