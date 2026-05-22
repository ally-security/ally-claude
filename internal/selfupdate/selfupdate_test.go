package selfupdate

import (
	"archive/tar"
	"archive/zip"
	"bytes"
	"compress/gzip"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func makeTarGz(t *testing.T, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: "claude-3p-helper", Mode: 0o755, Size: int64(len(body))}
	if err := tw.WriteHeader(hdr); err != nil {
		t.Fatal(err)
	}
	tw.Write(body)
	tw.Close()
	gz.Close()
	return buf.Bytes()
}

func makeZipBinary(t *testing.T, body []byte) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("claude-3p-helper.exe")
	w.Write(body)
	zw.Close()
	return buf.Bytes()
}

func sha(b []byte) string {
	s := sha256.Sum256(b)
	return hex.EncodeToString(s[:])
}

// fakeGithubServer wires up /releases/latest and asset routes.
func fakeGithubServer(t *testing.T, tag string, assets map[string][]byte) *httptest.Server {
	t.Helper()
	var srv *httptest.Server
	mux := http.NewServeMux()
	srv = httptest.NewServer(mux)
	mux.HandleFunc("/repos/", func(w http.ResponseWriter, r *http.Request) {
		// /repos/owner/name/releases/latest
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
		data, ok := assets[name]
		if !ok {
			http.NotFound(w, r)
			return
		}
		w.Write(data)
	})
	return srv
}

// withGithubBase points api.github.com calls at our test server by
// substituting it via Options.HTTPClient + a transport that rewrites.
func withGithubBase(srv *httptest.Server) *http.Client {
	return &http.Client{Transport: rewrite{base: srv.URL}}
}

type rewrite struct{ base string }

func (r rewrite) RoundTrip(req *http.Request) (*http.Response, error) {
	req.URL.Scheme = "http"
	req.URL.Host = strings.TrimPrefix(r.base, "http://")
	return http.DefaultTransport.RoundTrip(req)
}

func TestParseChecksum(t *testing.T) {
	body := "deadbeef  other.tgz\nabc123  asset.tgz\n"
	if got := parseChecksum(body, "asset.tgz"); got != "abc123" {
		t.Errorf("got %q", got)
	}
	if got := parseChecksum(body, "missing"); got != "" {
		t.Errorf("expected empty, got %q", got)
	}
}

func TestNormaliseVersion(t *testing.T) {
	if NormaliseVersion("v1.2.3") != "1.2.3" {
		t.Error("v prefix should be stripped")
	}
	if NormaliseVersion("1.2.3") != "1.2.3" {
		t.Error("non-prefixed should be unchanged")
	}
}

func TestVerifySHA256(t *testing.T) {
	if err := VerifySHA256([]byte("x"), ""); err != nil {
		t.Error("empty want should pass")
	}
	if err := VerifySHA256([]byte("x"), sha([]byte("x"))); err != nil {
		t.Error("matching should pass")
	}
	if err := VerifySHA256([]byte("x"), "deadbeef"); err == nil {
		t.Error("mismatch should fail")
	}
}

func TestPickAsset(t *testing.T) {
	rel := &Release{Assets: []Asset{
		{Name: "x_1.0_darwin_arm64.tar.gz"},
		{Name: "x_1.0_linux_amd64.tar.gz"},
		{Name: "x_1.0_windows_amd64.zip"},
		{Name: "checksums.txt"},
	}}
	if a, _ := PickAsset(rel, "darwin", "arm64"); a == nil || !strings.Contains(a.Name, "darwin_arm64") {
		t.Errorf("darwin pick wrong: %+v", a)
	}
	if a, _ := PickAsset(rel, "windows", "amd64"); a == nil || !strings.HasSuffix(a.Name, ".zip") {
		t.Errorf("windows pick wrong: %+v", a)
	}
	if _, err := PickAsset(rel, "plan9", "amd64"); err == nil {
		t.Error("expected error for unknown arch")
	}
}

func TestExtractBinaryTarGz(t *testing.T) {
	payload := []byte("BIN")
	tgz := makeTarGz(t, payload)
	got, err := ExtractBinary(tgz, "x_darwin_arm64.tar.gz")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("payload mismatch: %q", got)
	}
}

func TestExtractBinaryZip(t *testing.T) {
	payload := []byte("WIN")
	z := makeZipBinary(t, payload)
	got, err := ExtractBinary(z, "x_windows_amd64.zip")
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, payload) {
		t.Errorf("payload mismatch: %q", got)
	}
}

func TestExtractBinaryErrors(t *testing.T) {
	if _, err := ExtractBinary([]byte("garbage"), "x.tar.gz"); err == nil {
		t.Error("expected gzip error")
	}
	if _, err := ExtractBinary([]byte("garbage"), "x.zip"); err == nil {
		t.Error("expected zip error")
	}
	if _, err := ExtractBinary([]byte("x"), "unknown.rar"); err == nil {
		t.Error("expected unsupported error")
	}
	// tar.gz missing entry
	var buf bytes.Buffer
	gz := gzip.NewWriter(&buf)
	tw := tar.NewWriter(gz)
	hdr := &tar.Header{Name: "other", Mode: 0o644, Size: 1}
	tw.WriteHeader(hdr)
	tw.Write([]byte("x"))
	tw.Close()
	gz.Close()
	if _, err := ExtractBinary(buf.Bytes(), "x.tar.gz"); err == nil {
		t.Error("expected missing-entry error")
	}
	// zip missing entry
	var zbuf bytes.Buffer
	zw := zip.NewWriter(&zbuf)
	w, _ := zw.Create("other.exe")
	w.Write([]byte("x"))
	zw.Close()
	if _, err := ExtractBinary(zbuf.Bytes(), "x.zip"); err == nil {
		t.Error("expected missing-entry zip error")
	}
}

func TestReplaceBinary(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "bin")
	_ = os.WriteFile(target, []byte("old"), 0o755)
	if err := ReplaceBinary(target, []byte("new")); err != nil {
		t.Fatal(err)
	}
	got, _ := os.ReadFile(target)
	if string(got) != "new" {
		t.Errorf("content = %q", got)
	}
}

func TestFetchLatestAndChecksum(t *testing.T) {
	bin := []byte("BIN")
	tgz := makeTarGz(t, bin)
	checksums := fmt.Sprintf("%s  app_darwin_arm64.tar.gz\n", sha(tgz))
	srv := fakeGithubServer(t, "v1.2.3", map[string][]byte{
		"app_darwin_arm64.tar.gz": tgz,
		"checksums.txt":           []byte(checksums),
	})
	defer srv.Close()

	client := withGithubBase(srv)
	rel, err := FetchLatest(Options{Repo: "owner/name", HTTPClient: client})
	if err != nil {
		t.Fatal(err)
	}
	if rel.TagName != "v1.2.3" || len(rel.Assets) != 2 {
		t.Errorf("rel = %+v", rel)
	}
	sum, err := FetchChecksum(rel, "app_darwin_arm64.tar.gz", client)
	if err != nil {
		t.Fatal(err)
	}
	if sum != sha(tgz) {
		t.Errorf("sum = %q", sum)
	}
	// No checksums asset -> empty string.
	rel2 := &Release{Assets: nil}
	got, err := FetchChecksum(rel2, "x", client)
	if err != nil || got != "" {
		t.Errorf("expected empty, got %q %v", got, err)
	}
}

func TestFetchLatest404(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(404)
	}))
	defer srv.Close()
	_, err := FetchLatest(Options{Repo: "x/y", HTTPClient: withGithubBase(srv)})
	if err == nil {
		t.Error("expected 404 error")
	}
}

func TestDownload(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("ok"))
	}))
	defer srv.Close()
	data, err := Download(&Asset{DownloadURL: srv.URL}, nil)
	if err != nil || string(data) != "ok" {
		t.Errorf("data=%q err=%v", data, err)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer bad.Close()
	if _, err := Download(&Asset{DownloadURL: bad.URL}, nil); err == nil {
		t.Error("expected 500 error")
	}
}

func TestRunFullFlow(t *testing.T) {
	bin := []byte("NEW-BINARY")
	tgz := makeTarGz(t, bin)
	assetName := "x_1.2.3_darwin_arm64.tar.gz"
	checksums := fmt.Sprintf("%s  %s\n", sha(tgz), assetName)
	srv := fakeGithubServer(t, "v1.2.3", map[string][]byte{
		assetName:       tgz,
		"checksums.txt": []byte(checksums),
	})
	defer srv.Close()

	dir := t.TempDir()
	target := filepath.Join(dir, "claude-3p-helper")
	_ = os.WriteFile(target, []byte("old"), 0o755)

	res, err := Run(Options{
		Repo:       "x/y",
		Current:    "v1.0.0",
		GOOS:       "darwin",
		GOARCH:     "arm64",
		HTTPClient: withGithubBase(srv),
	}, target, false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.Replaced || res.Latest != "1.2.3" {
		t.Errorf("res = %+v", res)
	}
	got, _ := os.ReadFile(target)
	if !bytes.Equal(got, bin) {
		t.Errorf("binary not replaced: %q", got)
	}
}

func TestRunUpToDate(t *testing.T) {
	srv := fakeGithubServer(t, "v1.0.0", map[string][]byte{})
	defer srv.Close()
	res, err := Run(Options{Repo: "x/y", Current: "v1.0.0", HTTPClient: withGithubBase(srv)}, "/dev/null", false)
	if err != nil {
		t.Fatal(err)
	}
	if !res.UpToDate {
		t.Errorf("expected up-to-date, got %+v", res)
	}
}

func TestRunDryRun(t *testing.T) {
	bin := []byte("X")
	tgz := makeTarGz(t, bin)
	srv := fakeGithubServer(t, "v9.9.9", map[string][]byte{
		"x_9.9.9_linux_amd64.tar.gz": tgz,
	})
	defer srv.Close()
	res, err := Run(Options{
		Repo: "x/y", Current: "v1.0.0", GOOS: "linux", GOARCH: "amd64",
		HTTPClient: withGithubBase(srv),
	}, "/dev/null", true)
	if err != nil {
		t.Fatal(err)
	}
	if res.Replaced || res.Asset == "" {
		t.Errorf("dry run unexpected: %+v", res)
	}
}

func TestRunChecksumMismatch(t *testing.T) {
	tgz := makeTarGz(t, []byte("X"))
	asset := "x_9.9.9_linux_amd64.tar.gz"
	srv := fakeGithubServer(t, "v9.9.9", map[string][]byte{
		asset:           tgz,
		"checksums.txt": []byte("deadbeef  " + asset + "\n"),
	})
	defer srv.Close()
	_, err := Run(Options{
		Repo: "x/y", Current: "v1.0.0", GOOS: "linux", GOARCH: "amd64",
		HTTPClient: withGithubBase(srv),
	}, "/dev/null", false)
	if err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Errorf("expected mismatch err, got %v", err)
	}
}

func TestRunFetchLatestError(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer srv.Close()
	_, err := Run(Options{Repo: "x/y", Current: "v1", HTTPClient: withGithubBase(srv)}, "/x", false)
	if err == nil {
		t.Error("expected error")
	}
}

func TestFetchLatestBadJSON(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("not json"))
	}))
	defer srv.Close()
	_, err := FetchLatest(Options{Repo: "x/y", HTTPClient: withGithubBase(srv)})
	if err == nil {
		t.Error("expected decode error")
	}
}

func TestRunNoMatchingAsset(t *testing.T) {
	srv := fakeGithubServer(t, "v9.9.9", map[string][]byte{
		"only_windows_amd64.zip": []byte("x"),
	})
	defer srv.Close()
	_, err := Run(Options{
		Repo: "x/y", Current: "v1.0.0", GOOS: "darwin", GOARCH: "arm64",
		HTTPClient: withGithubBase(srv),
	}, "/dev/null", true)
	if err == nil {
		t.Error("expected no-asset error")
	}
}
