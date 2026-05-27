package slackmcp

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"
)

func SaveUserToken(token string) error {
	token = strings.TrimSpace(token)
	if token == "" {
		return fmt.Errorf("empty Slack user token")
	}
	return keychainSet(KeychainService, UserTokenAccount, []byte(token))
}

func LoadUserToken() (string, error) {
	if token, err := loadUserTokenFromKeychain(); err == nil {
		return token, nil
	}
	if token, err := importLegacyTokenFile(); err == nil {
		_ = SaveUserToken(token)
		return token, nil
	}
	return "", fmt.Errorf("no Slack user token — run: ally3p claude login [policy.yaml]")
}

func loadUserTokenFromKeychain() (string, error) {
	data, err := keychainGet(KeychainService, UserTokenAccount)
	if err != nil {
		return "", err
	}
	token := strings.TrimSpace(string(data))
	if token == "" {
		return "", fmt.Errorf("empty Slack token in Keychain")
	}
	return token, nil
}

func importLegacyTokenFile() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	path := filepath.Join(home, ".config", "claude-3p", "slack-mcp-token.json")
	data, err := os.ReadFile(path)
	if err != nil {
		return "", err
	}
	var payload struct {
		AccessToken string `json:"access_token"`
	}
	if err := json.Unmarshal(data, &payload); err != nil {
		return "", err
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return "", fmt.Errorf("legacy token file has no access_token")
	}
	return payload.AccessToken, nil
}

func HasUserToken() bool {
	_, err := LoadUserToken()
	return err == nil
}
