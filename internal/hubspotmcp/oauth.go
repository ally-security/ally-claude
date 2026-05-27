package hubspotmcp

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

const (
	authURL  = "https://mcp.hubspot.com/oauth/authorize/user"
	tokenURL = "https://mcp.hubspot.com/oauth/v3/token"
)

func EnsureLogin() error {
	if HasUserToken() {
		if _, err := AccessToken(); err == nil {
			return nil
		}
	}
	return Login()
}

func Login() error {
	creds, err := LoadClientCredentials()
	if err != nil {
		return err
	}
	redirectURI := fmt.Sprintf("http://127.0.0.1:%d/callback", creds.CallbackPort)
	verifier, challenge := pkcePair()
	authURL := buildAuthURL(creds.ClientID, redirectURI, challenge)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)
	mux := http.NewServeMux()
	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		if qerr := r.URL.Query().Get("error"); qerr != "" {
			desc := r.URL.Query().Get("error_description")
			if desc != "" {
				errCh <- fmt.Errorf("hubspot oauth error: %s (%s)", qerr, desc)
			} else {
				errCh <- fmt.Errorf("hubspot oauth error: %s", qerr)
			}
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		code := r.URL.Query().Get("code")
		if code == "" {
			errCh <- fmt.Errorf("missing authorization code")
			w.WriteHeader(http.StatusBadRequest)
			return
		}
		codeCh <- code
		w.Header().Set("Content-Type", "text/html")
		_, _ = w.Write([]byte("<html><body><h1>HubSpot connected</h1><p>You can close this tab.</p></body></html>"))
	})
	srv := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", creds.CallbackPort), Handler: mux}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer func() { _ = srv.Close() }()

	openBrowser(authURL)
	fmt.Fprintf(os.Stderr, "HubSpot sign-in: if the browser did not open:\n%s\n", authURL)
	fmt.Fprintf(os.Stderr, "Register redirect URI on your HubSpot MCP Auth App: %s\n", redirectURI)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("hubspot oauth callback timeout after 5m")
	}

	token, err := exchangeCode(creds, redirectURI, code, verifier)
	if err != nil {
		return err
	}
	if err := SaveUserToken(token); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "✓ saved HubSpot user token")
	return nil
}

func buildAuthURL(clientID, redirectURI, challenge string) string {
	v := url.Values{}
	v.Set("client_id", clientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("response_type", "code")
	v.Set("code_challenge", challenge)
	v.Set("code_challenge_method", "S256")
	return authURL + "?" + v.Encode()
}

func exchangeCode(creds ClientCredentials, redirectURI, code, verifier string) (UserToken, error) {
	form := url.Values{}
	form.Set("grant_type", "authorization_code")
	form.Set("client_id", creds.ClientID)
	form.Set("client_secret", creds.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", verifier)

	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return UserToken{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return UserToken{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return UserToken{}, fmt.Errorf("hubspot token exchange failed (%d): %s", resp.StatusCode, string(body))
	}
	return parseTokenResponse(body)
}

func refreshToken(creds ClientCredentials, refresh string) (UserToken, error) {
	form := url.Values{}
	form.Set("grant_type", "refresh_token")
	form.Set("client_id", creds.ClientID)
	form.Set("client_secret", creds.ClientSecret)
	form.Set("refresh_token", refresh)

	resp, err := http.PostForm(tokenURL, form)
	if err != nil {
		return UserToken{}, err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return UserToken{}, err
	}
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		return UserToken{}, fmt.Errorf("hubspot token refresh failed (%d): %s", resp.StatusCode, string(body))
	}
	return parseTokenResponse(body)
}

func parseTokenResponse(body []byte) (UserToken, error) {
	var payload struct {
		AccessToken  string `json:"access_token"`
		RefreshToken string `json:"refresh_token"`
		ExpiresIn    int    `json:"expires_in"`
	}
	if err := json.Unmarshal(body, &payload); err != nil {
		return UserToken{}, err
	}
	if strings.TrimSpace(payload.AccessToken) == "" {
		return UserToken{}, fmt.Errorf("hubspot token response missing access_token: %s", string(body))
	}
	expiresAt := time.Now().Add(time.Duration(payload.ExpiresIn) * time.Second)
	if payload.ExpiresIn == 0 {
		expiresAt = time.Now().Add(30 * time.Minute)
	}
	return UserToken{
		AccessToken:  payload.AccessToken,
		RefreshToken: payload.RefreshToken,
		ExpiresAt:    expiresAt,
	}, nil
}

func pkcePair() (verifier, challenge string) {
	buf := make([]byte, 32)
	_, _ = rand.Read(buf)
	verifier = base64.RawURLEncoding.EncodeToString(buf)
	sum := sha256.Sum256([]byte(verifier))
	challenge = base64.RawURLEncoding.EncodeToString(sum[:])
	return verifier, challenge
}

func openBrowser(u string) {
	switch runtime.GOOS {
	case "darwin":
		_ = exec.Command("open", u).Start()
	case "linux":
		_ = exec.Command("xdg-open", u).Start()
	case "windows":
		_ = exec.Command("rundll32", "url.dll,FileProtocolHandler", u).Start()
	}
}
