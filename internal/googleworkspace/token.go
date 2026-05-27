package googleworkspace

import (
	"encoding/json"
	"fmt"
	"time"
)

// Token is persisted in the OS credential store (Keychain on macOS).
type Token struct {
	ServiceID    string  `json:"service_id"`
	ClientID     string  `json:"client_id"`
	ClientSecret string  `json:"client_secret"`
	AccessToken  string  `json:"access_token"`
	RefreshToken string  `json:"refresh_token,omitempty"`
	ExpiresAt    float64 `json:"expires_at"`
}

func (t *Token) NeedsRefresh(skew time.Duration) bool {
	if t.AccessToken == "" {
		return true
	}
	return float64(time.Now().Unix()) >= t.ExpiresAt-skew.Seconds()
}

func (t *Token) Marshal() ([]byte, error) {
	return json.Marshal(t)
}

func ParseToken(data []byte) (*Token, error) {
	var t Token
	if err := json.Unmarshal(data, &t); err != nil {
		return nil, err
	}
	if t.ClientID == "" || t.ClientSecret == "" {
		return nil, fmt.Errorf("token blob missing client_id or client_secret")
	}
	return &t, nil
}
