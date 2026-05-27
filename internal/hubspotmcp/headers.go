package hubspotmcp

import (
	"encoding/json"
	"fmt"
	"os"
	"time"
)

func AccessToken() (string, error) {
	token, err := LoadUserToken()
	if err != nil {
		return "", err
	}
	if time.Until(token.ExpiresAt) > 60*time.Second {
		return token.AccessToken, nil
	}
	if token.RefreshToken == "" {
		return "", fmt.Errorf("hubspot access token expired — run: ally3p claude login [policy.yaml]")
	}
	creds, err := LoadClientCredentials()
	if err != nil {
		return "", err
	}
	refreshed, err := refreshToken(creds, token.RefreshToken)
	if err != nil {
		return "", err
	}
	if refreshed.RefreshToken == "" {
		refreshed.RefreshToken = token.RefreshToken
	}
	if err := SaveUserToken(refreshed); err != nil {
		return "", err
	}
	return refreshed.AccessToken, nil
}

func PrintHeadersJSON() error {
	token, err := AccessToken()
	if err != nil {
		return err
	}
	out, err := json.Marshal(map[string]string{
		"Authorization": "Bearer " + token,
	})
	if err != nil {
		return err
	}
	_, err = fmt.Fprintln(os.Stdout, string(out))
	return err
}
