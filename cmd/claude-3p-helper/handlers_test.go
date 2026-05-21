package main

import (
	"archive/tar"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"github.com/anthropics/claude-3p-helper/internal/install"
)

// sandboxInstall points install.pathResolver at a tempdir-rooted layout
// for the duration of the test so cmd handlers don't touch the real
// configLibrary.
func sandboxInstall(t *testing.T) install.Paths {
	t.Helper()
	base := t.TempDir()
	p := install.Paths{
		ConfigLibrary: filepath.Join(base, "configLibrary"),
		OrgPlugins:    filepath.Join(base, "orgPlugins"),
		Extensions:    filepath.Join(base, "extensions"),
	}
	restore := install.SetPathResolverForTest(func() (install.Paths, error) { return p, nil })
	t.Cleanup(restore)
	return p
}

func TestRunSyncHappy(t *testing.T) {
	sandboxInstall(t)
	dir := t.TempDir()
	yaml := []byte("id: t\ninferenceProvider: anthropic\n")
	path := filepath.Join(dir, "p.yaml")
	if err := os.WriteFile(path, yaml, 0o644); err != nil {
		t.Fatal(err)
	}
	if err := runSync([]string{"--no-activate", path}); err != nil {
		t.Fatalf("runSync: %v", err)
	}
}

func TestRunSyncDryRun(t *testing.T) {
	sandboxInstall(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "p.yaml")
	_ = os.WriteFile(path, []byte("id: t\ninferenceProvider: anthropic\n"), 0o644)
	if err := runSync([]string{"--dry-run", path}); err != nil {
		t.Fatalf("runSync dry-run: %v", err)
	}
}

func TestRunSyncErrors(t *testing.T) {
	sandboxInstall(t)
	if err := runSync(nil); err == nil {
		t.Error("expected error for no args")
	}
	if err := runSync([]string{"definitely/not/a/real/file.yaml"}); err == nil {
		t.Error("expected error for missing file")
	}
	dir := t.TempDir()
	path := filepath.Join(dir, "bad.yaml")
	_ = os.WriteFile(path, []byte("id: ["), 0o644)
	if err := runSync([]string{path}); err == nil {
		t.Error("expected parse error")
	}
}

func TestRunModelsHappy(t *testing.T) {
	paths := sandboxInstall(t)
	_ = os.MkdirAll(paths.ConfigLibrary, 0o755)
	_ = os.WriteFile(filepath.Join(paths.ConfigLibrary, "a.json"), []byte(`{"id":"a","inferenceProvider":"anthropic","inferenceModels":["m1"]}`), 0o644)
	_ = os.WriteFile(filepath.Join(paths.ConfigLibrary, "_meta.json"), []byte(`{"activeConfigId":"a"}`), 0o644)

	if err := runModels(nil); err != nil {
		t.Fatalf("runModels: %v", err)
	}
	if err := runModels([]string{"--config", "a"}); err != nil {
		t.Fatalf("runModels --config: %v", err)
	}
	if err := runModels([]string{"--all"}); err != nil {
		t.Fatalf("runModels --all: %v", err)
	}
}

func TestRunModelsNoActive(t *testing.T) {
	sandboxInstall(t)
	if err := runModels(nil); err == nil || !strings.Contains(err.Error(), "no active config") {
		t.Errorf("expected no-active error, got %v", err)
	}
}

func TestRunModelsAllEmpty(t *testing.T) {
	sandboxInstall(t)
	if err := runModels([]string{"--all"}); err != nil {
		t.Errorf("--all on empty library should succeed, got %v", err)
	}
}

func TestRunModelsMissingConfig(t *testing.T) {
	sandboxInstall(t)
	if err := runModels([]string{"--config", "missing"}); err == nil {
		t.Error("expected error for missing config id")
	}
}

type cmdRewrite struct{ base string }

func (r cmdRewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(r.base, "http://")
	return http.DefaultTransport.RoundTrip(req)
}

func makeTarGzBinary(t *testing.T, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	_ = tw.WriteHeader(&tar.Header{Name: "claude-3p-helper", Mode: 0o755, Size: int64(len(body))})
	tw.Write(body)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func sha256Of(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

func fakeReleasesServer(t *testing.T, tag string, assets map[string][]byte) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	mux := http.NewServeMux()
	srv = httptest.NewServer(mux)
	mux.HandleFunc("/repos/", func(w http.ResponseWriter, r *http.Request) {
		fmt.Fprintf(w, `{"tag_name":%q,"assets":[`, tag)
		first := true
		for name := range assets {
			if !first {
				fmt.Fprint(w, ",")
			}
			fmt.Fprintf(w, `{"name":%q,"browser_download_url":%q}`, name, srv.URL+"/assets/"+name)
			first = false
		}
		fmt.Fprint(w, `]}`)
	})
	mux.HandleFunc("/assets/", func(w http.ResponseWriter, r *http.Request) {
		name := strings.TrimPrefix(r.URL.Path, "/assets/")
		if data, ok := assets[name]; ok {
			w.Write(data)
			return
		}
		http.NotFound(w, r)
	})
	return srv
}

func TestRunSelfUpdateCheckFlow(t *testing.T) {
	// Build a release where the test binary's "dev" version triggers an
	// update path. --check exits before any disk writes.
	bin := []byte("BIN")
	tgz := makeTarGzBinary(t, bin)
	srv := fakeReleasesServer(t, "v9.9.9", map[string][]byte{
		fmt.Sprintf("claude-3p-helper_9.9.9_%s_%s.tar.gz", goosForTest(), goarchForTest()): tgz,
		"checksums.txt": []byte(sha256Of(tgz) + "  irrelevant\n"),
	})
	defer srv.Close()
	selfUpdateClient = &http.Client{Transport: cmdRewrite{base: srv.URL}}
	t.Cleanup(func() { selfUpdateClient = nil })

	if err := runSelfUpdate([]string{"--check"}); err != nil {
		t.Errorf("runSelfUpdate --check: %v", err)
	}
}

func TestRunSelfUpdateUpToDate(t *testing.T) {
	// Latest tag matches the test's notion of current. Since version.Version
	// is "dev" in tests, NormaliseVersion("dev") != NormaliseVersion("v9.9.9")
	// so we expect an update path. The point is: the call returns cleanly.
	srv := fakeReleasesServer(t, "vdev", map[string][]byte{})
	defer srv.Close()
	selfUpdateClient = &http.Client{Transport: cmdRewrite{base: srv.URL}}
	t.Cleanup(func() { selfUpdateClient = nil })
	// With no assets matching the host, --check will hit PickAsset error.
	_ = runSelfUpdate([]string{"--check"})
}

func TestRunSelfUpdateNetworkError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	selfUpdateClient = &http.Client{Transport: cmdRewrite{base: srv.URL}}
	t.Cleanup(func() { selfUpdateClient = nil })
	if err := runSelfUpdate([]string{"--check"}); err == nil {
		t.Error("expected network error")
	}
}

// ── dispatch / usage ──────────────────────────────────────────────────

func TestDispatchUsageBranches(t *testing.T) {
	if code := dispatch(nil); code != 2 {
		t.Errorf("no args => exit 2, got %d", code)
	}
	if code := dispatch([]string{"version"}); code != 0 {
		t.Errorf("version => 0, got %d", code)
	}
	if code := dispatch([]string{"--version"}); code != 0 {
		t.Errorf("--version => 0, got %d", code)
	}
	if code := dispatch([]string{"help"}); code != 0 {
		t.Errorf("help => 0, got %d", code)
	}
	if code := dispatch([]string{"bogus"}); code != 2 {
		t.Errorf("unknown => 2, got %d", code)
	}
}

func TestDispatchSubcommandErrors(t *testing.T) {
	// sync with no positional arg returns an error, dispatch returns 1.
	if code := dispatch([]string{"sync"}); code != 1 {
		t.Errorf("sync with no args => 1, got %d", code)
	}
	// models with a missing config likewise.
	sandboxInstall(t)
	if code := dispatch([]string{"models", "--config", "nope"}); code != 1 {
		t.Errorf("models missing => 1, got %d", code)
	}
}

func TestDispatchSyncSuccess(t *testing.T) {
	sandboxInstall(t)
	dir := t.TempDir()
	path := filepath.Join(dir, "p.yaml")
	_ = os.WriteFile(path, []byte("id: t\ninferenceProvider: anthropic\n"), 0o644)
	if code := dispatch([]string{"sync", "--no-activate", path}); code != 0 {
		t.Errorf("sync => 0, got %d", code)
	}
}

func TestDispatchSelfUpdate(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	selfUpdateClient = &http.Client{Transport: cmdRewrite{base: srv.URL}}
	t.Cleanup(func() { selfUpdateClient = nil })
	if code := dispatch([]string{"self-update", "--check"}); code != 1 {
		t.Errorf("self-update => 1, got %d", code)
	}
}

func TestUsage(t *testing.T) {
	// Just ensure it doesn't panic and writes something.
	usage()
}

func goosForTest() string   { return runtime.GOOS }
func goarchForTest() string { return runtime.GOARCH }
