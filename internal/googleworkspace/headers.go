package googleworkspace

import (
	"encoding/json"
	"errors"
	"os"
	"strings"
	"time"
)

func AuthorizationHeaders(store Store) (map[string]string, error) {
	tok, err := store.Load()
	if err != nil {
		return nil, err
	}

	if tok.NeedsRefresh(60 * time.Second) {
		refreshed, err := RefreshAccessToken(tok)
		if err != nil {
			return nil, err
		}
		if err := store.Save(refreshed); err != nil {
			return nil, err
		}
		tok = refreshed
	}

	return map[string]string{
		"Authorization": "Bearer " + tok.AccessToken,
	}, nil
}

// PrintHeadersJSON writes headers for Claude headersHelper. If the user has no token yet,
// it runs one shared browser login for all Google Workspace MCP services.
func PrintHeadersJSON(store Store, svc Service) error {
	userStore := SharedUserStore()
	headers, err := AuthorizationHeaders(userStore)
	if err == nil {
		return encodeHeaders(headers)
	}
	if !needsLogin(err) {
		return err
	}

	if err := EnsureUnifiedLogin(); err != nil {
		return err
	}
	headers, err = AuthorizationHeaders(userStore)
	if err != nil {
		return err
	}
	return encodeHeaders(headers)
}

func needsLogin(err error) bool {
	if err == nil {
		return false
	}
	msg := strings.ToLower(err.Error())
	return strings.Contains(msg, "no token") ||
		strings.Contains(msg, "sign in") ||
		strings.Contains(msg, "run login") ||
		strings.Contains(msg, "could not be found") ||
		errors.Is(err, errKeychainNotFound) ||
		errors.Is(err, os.ErrNotExist)
}

func encodeHeaders(headers map[string]string) error {
	enc := json.NewEncoder(os.Stdout)
	enc.SetEscapeHTML(false)
	return enc.Encode(headers)
}
