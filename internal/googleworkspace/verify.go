package googleworkspace

import (
	"fmt"
	"io"
	"net/http"
	"strings"
)

func VerifyRedirectURI(cfg Config) error {
	authURL := buildAuthURL(cfg, cfg.RedirectURI())
	res, err := http.Get(authURL)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	body, err := io.ReadAll(res.Body)
	if err != nil {
		return err
	}
	lower := strings.ToLower(string(body))
	if strings.Contains(lower, "redirect_uri_mismatch") {
		return fmt.Errorf("redirect_uri_mismatch: register %s in Google Cloud Console", cfg.RedirectURI())
	}
	if strings.Contains(lower, "invalid_client") {
		return fmt.Errorf("invalid_client: check GOOGLE_CLIENT_ID")
	}
	return nil
}

func VerifyMCPEndpoint(svc Service) error {
	req, err := http.NewRequest(http.MethodPost, svc.MCPURL, strings.NewReader(
		`{"jsonrpc":"2.0","method":"initialize","id":1,"params":{"protocolVersion":"2024-11-05","capabilities":{},"clientInfo":{"name":"google-workspace-mcp-auth","version":"1.0"}}}`,
	))
	if err != nil {
		return err
	}
	req.Header.Set("Content-Type", "application/json")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer res.Body.Close()
	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return fmt.Errorf("initialize returned HTTP %d", res.StatusCode)
	}
	return nil
}
