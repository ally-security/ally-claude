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
	_, _, err := Resolve("foo", "main")
	if err == nil || !strings.Contains(err.Error(), "not a local file") {
		t.Fatalf("expected parse error, got %v", err)
	}
}

func TestResolveGitHubFetchFails(t *testing.T) {
	// Three-segment arg with a host the resolver will try to reach.
	// The raw.githubusercontent.com path will 404 for a clearly-nonexistent
	// path; we just need the error wrapping to fire.
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
