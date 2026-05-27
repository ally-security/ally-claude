package slackmcp

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
	DefaultCallbackPort = 3118
	MCPURL              = "https://mcp.slack.com/mcp"
)

var defaultUserScopes = []string{
	"chat:write",
	"search:read.public",
	"search:read.private",
	"search:read.im",
	"search:read.mpim",
	"search:read.files",
	"search:read.users",
	"channels:history",
	"groups:history",
	"im:history",
	"mpim:history",
	"users:read",
	"users:read.email",
}

func EnsureLogin() error {
	if HasUserToken() {
		return nil
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
			errCh <- fmt.Errorf("slack oauth error: %s", qerr)
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
		_, _ = w.Write([]byte("<html><body><h1>Slack connected</h1><p>You can close this tab.</p></body></html>"))
	})
	srv := &http.Server{Addr: fmt.Sprintf("127.0.0.1:%d", creds.CallbackPort), Handler: mux}

	go func() {
		if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
			errCh <- err
		}
	}()
	defer func() { _ = srv.Close() }()

	openBrowser(authURL)
	fmt.Fprintf(os.Stderr, "Slack sign-in: if the browser did not open:\n%s\n", authURL)
	fmt.Fprintf(os.Stderr, "Register redirect URI on your Slack app: %s\n", redirectURI)

	var code string
	select {
	case code = <-codeCh:
	case err := <-errCh:
		return err
	case <-time.After(5 * time.Minute):
		return fmt.Errorf("slack oauth callback timeout after 5m")
	}

	token, err := exchangeCode(creds, redirectURI, code, verifier)
	if err != nil {
		return err
	}
	if err := SaveUserToken(token); err != nil {
		return err
	}
	fmt.Fprintln(os.Stderr, "✓ saved Slack user token")
	return nil
}

func buildAuthURL(clientID, redirectURI, challenge string) string {
	v := url.Values{}
	v.Set("client_id", clientID)
	v.Set("redirect_uri", redirectURI)
	v.Set("response_type", "code")
	v.Set("scope", strings.Join(defaultUserScopes, " "))
	v.Set("code_challenge", challenge)
	v.Set("code_challenge_method", "S256")
	return "https://slack.com/oauth/v2_user/authorize?" + v.Encode()
}

func exchangeCode(creds ClientCredentials, redirectURI, code, verifier string) (string, error) {
	form := url.Values{}
	form.Set("client_id", creds.ClientID)
	form.Set("client_secret", creds.ClientSecret)
	form.Set("code", code)
	form.Set("redirect_uri", redirectURI)
	form.Set("code_verifier", verifier)

	resp, err := http.PostForm("https://slack.com/api/oauth.v2.user.access", form)
	if err != nil {
		return "", err
	}
	defer resp.Body.Close()
	body, err := io.ReadAll(resp.Body)
	if err != nil {
		return "", err
	}
	var payload map[string]interface{}
	if err := json.Unmarshal(body, &payload); err != nil {
		return "", err
	}
	if ok, _ := payload["ok"].(bool); !ok {
		return "", fmt.Errorf("slack token exchange failed: %s", string(body))
	}
	if token, _ := payload["access_token"].(string); token != "" {
		return token, nil
	}
	if authed, _ := payload["authed_user"].(map[string]interface{}); authed != nil {
		if token, _ := authed["access_token"].(string); token != "" {
			return token, nil
		}
	}
	return "", fmt.Errorf("slack token exchange returned no access_token: %s", string(body))
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
