package googleworkspace

import (
	"os"
	"strconv"
)

// Config holds OAuth client settings for one Workspace MCP service.
type Config struct {
	Service      Service
	ClientID     string
	ClientSecret string
	CallbackPort int
	Scope        string
}

func LoadConfig(svc Service) (Config, error) {
	creds, err := LoadClientCredentials(svc)
	if err != nil {
		return Config{}, err
	}
	return mergeCredentialsIntoConfig(svc, creds), nil
}

func (c Config) RedirectURI() string {
	return "http://127.0.0.1:" + strconv.Itoa(c.CallbackPort) + "/callback"
}

func KeychainServiceNameForStore() string {
	if s := os.Getenv("GOOGLE_MCP_KEYCHAIN_SERVICE"); s != "" {
		return s
	}
	return KeychainServiceName
}
