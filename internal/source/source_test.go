package source

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestResolveLocalDirect(t *testing.T) {
	dir := t.TempDir()
	p := filepath.Join(dir, "policy.yaml")
	if err := os.WriteFile(p, []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	data, origin, err := Resolve(p, "main")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "hi" || !strings.HasPrefix(origin, "local:") {
		t.Errorf("data=%q origin=%q", data, origin)
	}
}

func TestResolveTildeExpansion(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)
	if err := os.WriteFile(filepath.Join(home, "p.yaml"), []byte("hi"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, origin, err := Resolve("~/p.yaml", "main")
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasPrefix(origin, "local:") {
		t.Errorf("origin=%q", origin)
	}
}

func TestResolveGitHubFallbackParseError(t *testing.T) {
	// Make sure no inherited token forces the API branch.
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	_, _, err := Resolve("foo", "main")
	if err == nil || !strings.Contains(err.Error(), "not a local file") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestResolveGitHubFetchFails(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	_, _, err := Resolve("anthropics/__no_such_repo__/file.yaml", "main")
	if err == nil {
		t.Fatal("expected error from github fetch")
	}
	if !strings.Contains(err.Error(), "github fetch failed") {
		t.Errorf("err = %v", err)
	}
}

func TestFetchHTTPOK(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()
	data, err := fetchHTTP(srv.URL)
	if err != nil || string(data) != "ok" {
		t.Errorf("data=%q err=%v", data, err)
	}
}

func TestFetchHTTPErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusNotFound)
	}))
	defer srv.Close()
	if _, err := fetchHTTP(srv.URL); err == nil {
		t.Error("expected error on 404")
	}
	if _, err := fetchHTTP("http://127.0.0.1:1"); err == nil {
		t.Error("expected dial error")
	}
}

func TestFetchAPISendsAuthAndAccept(t *testing.T) {
	var gotAuth, gotAccept string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		gotAuth = r.Header.Get("Authorization")
		gotAccept = r.Header.Get("Accept")
		w.Write([]byte("private-bytes"))
	}))
	defer srv.Close()

	data, err := fetchAPI(srv.URL, "secret-token")
	if err != nil {
		t.Fatal(err)
	}
	if string(data) != "private-bytes" {
		t.Errorf("body=%q", data)
	}
	if gotAuth != "token secret-token" {
		t.Errorf("Authorization header = %q", gotAuth)
	}
	if gotAccept != "application/vnd.github.raw" {
		t.Errorf("Accept header = %q", gotAccept)
	}
}

func TestFetchAPIErrors(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer srv.Close()
	if _, err := fetchAPI(srv.URL, "t"); err == nil {
		t.Error("expected 401 to error")
	}
}

// resolveWithBaseURL exercises Resolve's API path by pointing the
// resolver at a local test server via a custom HTTP transport. We do
// this by monkey-patching net/http's DefaultTransport — but instead,
// since Resolve constructs URLs with the live github host, we test the
// token-aware branching by checking that the *path* selection happens
// correctly given GITHUB_TOKEN, even when the fetch itself fails.
func TestResolveTokenPathSelected(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "fake-token-for-test")
	t.Cleanup(func() { _ = os.Unsetenv("GITHUB_TOKEN") })
	_, _, err := Resolve("anthropics/__no_such_repo__/file.yaml", "main")
	if err == nil {
		t.Fatal("expected error from github API fetch")
	}
	if !strings.Contains(err.Error(), "github API fetch failed") {
		t.Errorf("expected API error, got %v", err)
	}
}

func TestGithubTokenPrecedence(t *testing.T) {
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "from-gh")
	if got := githubToken(); got != "from-gh" {
		t.Errorf("got %q, want from-gh", got)
	}
	t.Setenv("GITHUB_TOKEN", "from-github")
	if got := githubToken(); got != "from-github" {
		t.Errorf("got %q, want from-github (precedence)", got)
	}
	t.Setenv("GITHUB_TOKEN", "")
	t.Setenv("GH_TOKEN", "")
	if got := githubToken(); got != "" {
		t.Errorf("got %q, want empty", got)
	}
}
