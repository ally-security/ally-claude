package googleworkspace

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"os/exec"
	"runtime"
	"strings"
	"time"
)

type tokenResponse struct {
	AccessToken  string `json:"access_token"`
	TokenType    string `json:"token_type"`
	ExpiresIn    int    `json:"expires_in"`
	RefreshToken string `json:"refresh_token"`
	Error        string `json:"error"`
	ErrorDesc    string `json:"error_description"`
}

// Login runs the browser OAuth loop and saves tokens to the keychain.
func Login(cfg Config, store Store) error {
	code, err := listenForAuthCode(cfg)
	if err != nil {
		return err
	}

	tok, err := exchangeCode(cfg, code)
	if err != nil {
		return err
	}

	access := tok.AccessToken
	if access == "" {
		return fmt.Errorf("token response missing access_token: %s", tok.ErrorDesc)
	}

	now := time.Now()
	record := &Token{
		ServiceID:    cfg.Service.ID,
		ClientID:     cfg.ClientID,
		ClientSecret: cfg.ClientSecret,
		AccessToken:  access,
		RefreshToken: tok.RefreshToken,
		ExpiresAt:    float64(now.Unix()) + float64(tok.ExpiresIn),
	}
	if record.RefreshToken == "" {
		return fmt.Errorf("no refresh_token returned; use prompt=consent and access_type=offline")
	}

	return store.Save(record)
}

func listenForAuthCode(cfg Config) (string, error) {
	redirect := cfg.RedirectURI()
	authURL := buildAuthURL(cfg, redirect)

	codeCh := make(chan string, 1)
	errCh := make(chan error, 1)

	mux := http.NewServeMux()
	server := &http.Server{Handler: mux}

	mux.HandleFunc("/callback", func(w http.ResponseWriter, r *http.Request) {
		q := r.URL.Query()
		if errMsg := q.Get("error"); errMsg != "" {
			errCh <- fmt.Errorf("oauth error: %s (%s)", errMsg, q.Get("error_description"))
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Authorization failed. You can close this tab."))
			return
		}
		code := q.Get("code")
		if code == "" {
			errCh <- fmt.Errorf("callback missing code")
			w.WriteHeader(http.StatusBadRequest)
			_, _ = w.Write([]byte("Missing authorization code."))
			return
		}
		codeCh <- code
		w.WriteHeader(http.StatusOK)
		_, _ = w.Write([]byte("OAuth OK. You can close this tab and return to the terminal."))
	})

	ln, err := net.Listen("tcp", fmt.Sprintf("127.0.0.1:%d", cfg.CallbackPort))
	if err != nil {
		return "", fmt.Errorf("listen on %s: %w", redirect, err)
	}

	go func() {
		_ = server.Serve(ln)
	}()
	defer func() {
		ctx, cancel := context.WithTimeout(context.Background(), 2*time.Second)
		defer cancel()
		_ = server.Shutdown(ctx)
	}()

	_ = openBrowser(authURL)
	fmt.Fprintf(stderr, "[%s] listening on %s\nIf the browser did not open:\n%s\n", cfg.Service.ID, redirect, authURL)

	select {
	case code := <-codeCh:
		return code, nil
	case err := <-errCh:
		return "", err
	case <-time.After(5 * time.Minute):
		return "", fmt.Errorf("oauth callback timeout after 5m")
	}
}

func buildAuthURL(cfg Config, redirect string) string {
	v := url.Values{}
	v.Set("client_id", cfg.ClientID)
	v.Set("redirect_uri", redirect)
	v.Set("response_type", "code")
	v.Set("scope", cfg.Scope)
	v.Set("access_type", "offline")
	v.Set("prompt", "consent")
	return "https://accounts.google.com/o/oauth2/v2/auth?" + v.Encode()
}

func exchangeCode(cfg Config, code string) (*tokenResponse, error) {
	return postToken(url.Values{
		"code":          {code},
		"client_id":     {cfg.ClientID},
		"client_secret": {cfg.ClientSecret},
		"redirect_uri":  {cfg.RedirectURI()},
		"grant_type":    {"authorization_code"},
	})
}

// RefreshAccessToken exchanges refresh_token for a new access_token.
func RefreshAccessToken(t *Token) (*Token, error) {
	if t.RefreshToken == "" {
		return nil, fmt.Errorf("no refresh_token; run login again")
	}
	resp, err := postToken(url.Values{
		"client_id":     {t.ClientID},
		"client_secret": {t.ClientSecret},
		"refresh_token": {t.RefreshToken},
		"grant_type":    {"refresh_token"},
	})
	if err != nil {
		return nil, err
	}
	out := *t
	out.AccessToken = resp.AccessToken
	out.ExpiresAt = float64(time.Now().Unix()) + float64(resp.ExpiresIn)
	if resp.RefreshToken != "" {
		out.RefreshToken = resp.RefreshToken
	}
	return &out, nil
}

func postToken(form url.Values) (*tokenResponse, error) {
	req, err := http.NewRequest(http.MethodPost, "https://oauth2.googleapis.com/token", strings.NewReader(form.Encode()))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")

	res, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer res.Body.Close()

	body, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, err
	}
	var tok tokenResponse
	if err := json.Unmarshal(body, &tok); err != nil {
		return nil, err
	}
	if res.StatusCode >= 400 || tok.Error != "" {
		msg := tok.ErrorDesc
		if msg == "" {
			msg = string(body)
		}
		return nil, fmt.Errorf("token endpoint HTTP %d: %s", res.StatusCode, msg)
	}
	return &tok, nil
}

func openBrowser(u string) error {
	var cmd *exec.Cmd
	switch runtime.GOOS {
	case "darwin":
		cmd = exec.Command("open", u)
	case "windows":
		cmd = exec.Command("cmd", "/c", "start", u)
	default:
		cmd = exec.Command("xdg-open", u)
	}
	return cmd.Start()
}

var stderr io.Writer = io.Discard

func SetStderr(w io.Writer) {
	stderr = w
}
