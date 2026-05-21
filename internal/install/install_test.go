package install

import (
	"archive/zip"
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/anthropics/claude-3p-helper/internal/policy"
)

// withSandboxPaths swaps pathResolver to a tempdir-rooted layout for the
// duration of the test.
func withSandboxPaths(t *testing.T) Paths {
	t.Helper()
	base := t.TempDir()
	p := layout(base)
	prev := pathResolver
	pathResolver = func() (Paths, error) { return p, nil }
	t.Cleanup(func() { pathResolver = prev })
	return p
}

func boolP(b bool) *bool { return &b }

// ── paths.go ──────────────────────────────────────────────────────────

func TestResolvePathsFor(t *testing.T) {
	t.Setenv("HOME", "/h")
	t.Setenv("LOCALAPPDATA", "/app")

	cases := map[string]struct {
		goos    string
		wantSub string
		wantErr bool
	}{
		"darwin":  {"darwin", "Library/Application Support/Claude-3p/configLibrary", false},
		"linux":   {"linux", ".config/Claude-3p/configLibrary", false},
		"windows": {"windows", "Claude-3p", false},
		"plan9":   {"plan9", "", true},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			p, err := ResolvePathsFor(tc.goos)
			if (err != nil) != tc.wantErr {
				t.Fatalf("err=%v wantErr=%v", err, tc.wantErr)
			}
			if !tc.wantErr && !strings.Contains(p.ConfigLibrary, tc.wantSub) {
				t.Errorf("ConfigLibrary=%q want substring %q", p.ConfigLibrary, tc.wantSub)
			}
		})
	}
}

func TestResolvePathsWindowsMissingEnv(t *testing.T) {
	t.Setenv("LOCALAPPDATA", "")
	if _, err := ResolvePathsFor("windows"); err == nil {
		t.Error("expected error when LOCALAPPDATA unset")
	}
}

// ── config.go ─────────────────────────────────────────────────────────

func TestBuildConfigDocPopulates(t *testing.T) {
	tru := true
	c := &policy.Config{
		ID:                           "abc",
		Name:                         "Acme",
		InferenceProvider:            "bedrock",
		InferenceCustomHeaders:       map[string]string{"H": "v"},
		InferenceAnthropicApiKey:     "k",
		InferenceCredentialHelper:    "/bin/helper",
		InferenceCredentialHelperTTL: 60,
		InferenceBedrockRegion:       "us-west-2",
		InferenceBedrockBearerToken:  "tok",
		InferenceBedrockProfile:      "p",
		InferenceBedrockSsoStartUrl:  "u",
		InferenceBedrockSsoRegion:    "r",
		InferenceBedrockSsoAccountId: "a",
		InferenceBedrockSsoRoleName:  "n",
		ModelDiscoveryEnabled:        &tru,
		InferenceModels:              []interface{}{"m1"},
		AllowedWorkspaceFolders:      []string{"~/x"},
		CoworkEgressAllowedHosts:     []string{"a.com"},
		DisabledBuiltinTools:         []string{"WebSearch"},
		BuiltinToolPolicy:            map[string]string{"Bash": "ask"},
		DeploymentOrganizationUUID:   "uuid",
		DisableEssentialTelemetry:    &tru,
		DisableNonessentialTelemetry: &tru,
		DisableAutoUpdates:           &tru,
		OTLPEndpoint:                 "https://e",
		OTLPProtocol:                 "grpc",
		OTLPHeaders:                  map[string]string{"H": "v"},
		IsLocalDevMcpEnabled:         boolP(false),
		IsDesktopExtensionEnabled:    boolP(true),
		OrgPluginSettings:            map[string]interface{}{"x": 1},
		Connectors:                   []policy.Connector{{Name: "c1", URL: "https://c"}},
	}
	doc := buildConfigDoc(c)
	for _, k := range []string{
		"id", "name", "inferenceProvider", "inferenceCustomHeaders",
		"inferenceBedrockRegion", "inferenceBedrockBearerToken",
		"modelDiscoveryEnabled", "inferenceModels",
		"allowedWorkspaceFolders", "coworkEgressAllowedHosts",
		"disabledBuiltinTools", "builtinToolPolicy",
		"deploymentOrganizationUuid", "disableEssentialTelemetry",
		"otlpEndpoint", "otlpProtocol", "otlpHeaders",
		"isLocalDevMcpEnabled", "isDesktopExtensionEnabled",
		"orgPluginSettings", "managedMcpServers",
	} {
		if _, ok := doc[k]; !ok {
			t.Errorf("doc missing %q", k)
		}
	}
}

func TestBuildConfigDocMinimal(t *testing.T) {
	c := &policy.Config{ID: "x", InferenceProvider: "anthropic"}
	doc := buildConfigDoc(c)
	if doc["id"] != "x" || doc["inferenceProvider"] != "anthropic" {
		t.Errorf("doc=%v", doc)
	}
	if len(doc) != 2 {
		t.Errorf("expected exactly 2 keys, got %d: %v", len(doc), doc)
	}
}

func TestBoolPtr(t *testing.T) {
	if boolPtr(nil) != nil {
		t.Error("nil pointer should produce nil interface")
	}
	if boolPtr(boolP(true)) != true {
		t.Error("expected true")
	}
}

func TestWriteAndActivateConfig(t *testing.T) {
	dir := t.TempDir()
	path, n, err := writeConfig(dir, "abc", map[string]interface{}{"k": "v"})
	if err != nil {
		t.Fatal(err)
	}
	if !strings.HasSuffix(path, "abc.json") || n <= 0 {
		t.Errorf("path=%q n=%d", path, n)
	}
	data, _ := os.ReadFile(path)
	var got map[string]interface{}
	_ = json.Unmarshal(data, &got)
	if got["k"] != "v" {
		t.Errorf("round-trip lost data: %v", got)
	}

	meta, err := activateConfig(dir, "abc")
	if err != nil {
		t.Fatal(err)
	}
	mdata, _ := os.ReadFile(meta)
	var m map[string]interface{}
	_ = json.Unmarshal(mdata, &m)
	if m["activeConfigId"] != "abc" {
		t.Errorf("active not set: %v", m)
	}

	// Activating again preserves other keys in _meta.json.
	_ = os.WriteFile(meta, []byte(`{"keep":"yes","activeConfigId":"old"}`), 0o644)
	_, _ = activateConfig(dir, "new")
	mdata, _ = os.ReadFile(meta)
	_ = json.Unmarshal(mdata, &m)
	if m["keep"] != "yes" || m["activeConfigId"] != "new" {
		t.Errorf("unexpected meta: %v", m)
	}
}

// ── fetch.go ──────────────────────────────────────────────────────────

func TestFetchHTTPAndFile(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("payload"))
	}))
	defer srv.Close()

	got, err := fetch(srv.URL)
	if err != nil || string(got) != "payload" {
		t.Errorf("http fetch: %q %v", got, err)
	}

	f := filepath.Join(t.TempDir(), "f")
	_ = os.WriteFile(f, []byte("local"), 0o644)
	got, err = fetch(f)
	if err != nil || string(got) != "local" {
		t.Errorf("file fetch: %q %v", got, err)
	}

	bad := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(500)
	}))
	defer bad.Close()
	if _, err := fetch(bad.URL); err == nil {
		t.Error("expected error on 500")
	}
}

func TestVerifySHA256(t *testing.T) {
	data := []byte("hello")
	sum := sha256.Sum256(data)
	hexsum := hex.EncodeToString(sum[:])
	if err := verifySHA256(data, ""); err != nil {
		t.Error("empty want should pass")
	}
	if err := verifySHA256(data, hexsum); err != nil {
		t.Error("matching sha should pass")
	}
	if err := verifySHA256(data, "deadbeef"); err == nil {
		t.Error("mismatching sha should fail")
	}
	if sha256Hex(data) != hexsum {
		t.Error("sha256Hex returned unexpected value")
	}
}

// ── plugins.go ────────────────────────────────────────────────────────

func makeZip(t *testing.T, files map[string]string) []byte {
	t.Helper()
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	for name, body := range files {
		w, err := zw.Create(name)
		if err != nil {
			t.Fatal(err)
		}
		w.Write([]byte(body))
	}
	if err := zw.Close(); err != nil {
		t.Fatal(err)
	}
	return buf.Bytes()
}

func TestInstallPluginAndIdempotency(t *testing.T) {
	zipBytes := makeZip(t, map[string]string{"manifest.json": "{}", "a/b.txt": "hi"})
	sum := sha256Hex(zipBytes)

	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(zipBytes)
	}))
	defer srv.Close()

	base := t.TempDir()
	bundle := policy.Bundle{Name: "p1", Source: srv.URL, SHA256: sum}

	res, err := installPlugin(base, bundle)
	if err != nil {
		t.Fatalf("installPlugin: %v", err)
	}
	if res.Skipped || res.Bytes != len(zipBytes) {
		t.Errorf("first install unexpected: %+v", res)
	}
	if _, err := os.Stat(filepath.Join(res.Dest, "manifest.json")); err != nil {
		t.Errorf("manifest not extracted: %v", err)
	}

	res2, err := installPlugin(base, bundle)
	if err != nil {
		t.Fatal(err)
	}
	if !res2.Skipped {
		t.Errorf("second install should skip, got %+v", res2)
	}
}

func TestInstallPluginShaMismatch(t *testing.T) {
	zipBytes := makeZip(t, map[string]string{"x": "y"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(zipBytes)
	}))
	defer srv.Close()

	_, err := installPlugin(t.TempDir(), policy.Bundle{Name: "p", Source: srv.URL, SHA256: "deadbeef"})
	if err == nil || !strings.Contains(err.Error(), "sha256 mismatch") {
		t.Errorf("expected sha mismatch error, got %v", err)
	}
}

func TestUnzipBytesRejectsZipSlip(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	w, _ := zw.Create("../escape.txt")
	w.Write([]byte("nope"))
	zw.Close()
	err := unzipBytes(buf.Bytes(), t.TempDir())
	if err == nil || !strings.Contains(err.Error(), "unsafe path") {
		t.Errorf("expected zip slip rejection, got %v", err)
	}
}

func TestInstallPluginNoSHA(t *testing.T) {
	zipBytes := makeZip(t, map[string]string{"f": "x"})
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(zipBytes)
	}))
	defer srv.Close()
	res, err := installPlugin(t.TempDir(), policy.Bundle{Name: "p", Source: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if res.SHA256 == "" {
		t.Error("expected SHA256 to be computed when not provided")
	}
}

// ── extensions.go ─────────────────────────────────────────────────────

func TestInstallExtensionAndIdempotency(t *testing.T) {
	payload := []byte("mcpb-contents")
	sum := sha256Hex(payload)
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write(payload)
	}))
	defer srv.Close()

	base := t.TempDir()
	b := policy.Bundle{Name: "ext", Source: srv.URL, SHA256: sum}
	res, err := installExtension(base, b)
	if err != nil {
		t.Fatal(err)
	}
	if res.Skipped || !strings.HasSuffix(res.Dest, "ext.mcpb") {
		t.Errorf("unexpected: %+v", res)
	}

	res2, _ := installExtension(base, b)
	if !res2.Skipped {
		t.Errorf("second should skip: %+v", res2)
	}
}

func TestInstallExtensionNoSHA(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("body"))
	}))
	defer srv.Close()
	res, err := installExtension(t.TempDir(), policy.Bundle{Name: "ext", Source: srv.URL})
	if err != nil {
		t.Fatal(err)
	}
	if res.SHA256 == "" {
		t.Error("expected computed sha")
	}
}

func TestInstallExtensionFetchError(t *testing.T) {
	_, err := installExtension(t.TempDir(), policy.Bundle{Name: "ext", Source: "http://127.0.0.1:1"})
	if err == nil {
		t.Error("expected dial error")
	}
}

func TestInstallPluginFetchError(t *testing.T) {
	_, err := installPlugin(t.TempDir(), policy.Bundle{Name: "p", Source: "http://127.0.0.1:1"})
	if err == nil {
		t.Error("expected dial error")
	}
}

func TestUnzipBytesDirectoryEntry(t *testing.T) {
	var buf bytes.Buffer
	zw := zip.NewWriter(&buf)
	// Explicit directory header with executable mode so the contained
	// file remains writable.
	dh := &zip.FileHeader{Name: "sub/", Method: zip.Store}
	dh.SetMode(os.ModeDir | 0o755)
	if _, err := zw.CreateHeader(dh); err != nil {
		t.Fatal(err)
	}
	w, _ := zw.Create("sub/x.txt")
	w.Write([]byte("hi"))
	zw.Close()
	dir := t.TempDir()
	if err := unzipBytes(buf.Bytes(), dir); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, "sub", "x.txt")); err != nil {
		t.Errorf("file not extracted: %v", err)
	}
}

func TestInstallExtensionShaMismatch(t *testing.T) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, _ *http.Request) {
		w.Write([]byte("x"))
	}))
	defer srv.Close()
	_, err := installExtension(t.TempDir(), policy.Bundle{Name: "e", Source: srv.URL, SHA256: "deadbeef"})
	if err == nil {
		t.Error("expected mismatch error")
	}
}

// ── plan.go ───────────────────────────────────────────────────────────

func TestPlanApplyEndToEnd(t *testing.T) {
	paths := withSandboxPaths(t)

	zipBytes := makeZip(t, map[string]string{"m.json": "{}"})
	extBytes := []byte("ext-bytes")
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/p.zip":
			w.Write(zipBytes)
		case "/e.mcpb":
			w.Write(extBytes)
		default:
			w.WriteHeader(404)
		}
	}))
	defer srv.Close()

	cfg := &policy.Config{
		ID:                "demo",
		InferenceProvider: "anthropic",
		Connectors:        []policy.Connector{{Name: "n", URL: "https://n"}},
		Plugins:           []policy.Bundle{{Name: "p", Source: srv.URL + "/p.zip"}},
		Extensions:        []policy.Bundle{{Name: "e", Source: srv.URL + "/e.mcpb"}},
	}
	plan, err := New(cfg, Options{Activate: true})
	if err != nil {
		t.Fatal(err)
	}
	var sb strings.Builder
	plan.Print(&sb)
	if !strings.Contains(sb.String(), "demo.json") || !strings.Contains(sb.String(), "activate:   yes") {
		t.Errorf("Print output unexpected:\n%s", sb.String())
	}

	if err := plan.Apply(); err != nil {
		t.Fatalf("Apply: %v", err)
	}
	if _, err := os.Stat(filepath.Join(paths.ConfigLibrary, "demo.json")); err != nil {
		t.Error("config not written")
	}
	if _, err := os.Stat(filepath.Join(paths.OrgPlugins, "p", "m.json")); err != nil {
		t.Error("plugin not extracted")
	}
	if _, err := os.Stat(filepath.Join(paths.Extensions, "e.mcpb")); err != nil {
		t.Error("extension not written")
	}
}

func TestPlanApplyNoActivate(t *testing.T) {
	withSandboxPaths(t)
	cfg := &policy.Config{ID: "x", InferenceProvider: "anthropic"}
	plan, _ := New(cfg, Options{Activate: false})
	if err := plan.Apply(); err != nil {
		t.Fatal(err)
	}
}

func TestPlanPrintStdioConnector(t *testing.T) {
	withSandboxPaths(t)
	cfg := &policy.Config{
		ID:                "x",
		InferenceProvider: "anthropic",
		Connectors:        []policy.Connector{{Name: "s", Command: "/bin/x"}, {Name: "u"}},
	}
	plan, _ := New(cfg, Options{Activate: false})
	var sb strings.Builder
	plan.Print(&sb)
	if !strings.Contains(sb.String(), "stdio:/bin/x") || !strings.Contains(sb.String(), "unknown") {
		t.Errorf("Print output unexpected:\n%s", sb.String())
	}
}

func TestPlanNewUnsupportedOS(t *testing.T) {
	prev := pathResolver
	pathResolver = func() (Paths, error) { return Paths{}, os.ErrNotExist }
	t.Cleanup(func() { pathResolver = prev })

	_, err := New(&policy.Config{ID: "x", InferenceProvider: "anthropic"}, Options{})
	if err == nil {
		t.Error("expected error from path resolver")
	}
}

// ── read.go ───────────────────────────────────────────────────────────

func TestReadConfigAndListAndActive(t *testing.T) {
	paths := withSandboxPaths(t)
	// No configs yet.
	ids, err := ListConfigs()
	if err != nil || ids != nil {
		t.Errorf("expected empty list, got %v %v", ids, err)
	}
	active, _, _ := ActiveID()
	if active != "" {
		t.Errorf("expected no active id, got %q", active)
	}

	// Write two configs.
	if _, _, err := writeConfig(paths.ConfigLibrary, "a", map[string]interface{}{
		"id":                     "a",
		"inferenceProvider":      "bedrock",
		"inferenceBedrockRegion": "us-west-2",
		"inferenceModels":        []interface{}{"m1", map[string]interface{}{"name": "m2", "labelOverride": "Two", "supports1m": true}},
	}); err != nil {
		t.Fatal(err)
	}
	if _, _, err := writeConfig(paths.ConfigLibrary, "b", map[string]interface{}{
		"id": "b", "inferenceProvider": "anthropic",
	}); err != nil {
		t.Fatal(err)
	}
	if _, err := activateConfig(paths.ConfigLibrary, "a"); err != nil {
		t.Fatal(err)
	}

	ids, _ = ListConfigs()
	if len(ids) != 2 || ids[0] != "a" || ids[1] != "b" {
		t.Errorf("ListConfigs() = %v", ids)
	}
	active, _, _ = ActiveID()
	if active != "a" {
		t.Errorf("active = %q", active)
	}

	c, err := LoadConfig("a")
	if err != nil {
		t.Fatal(err)
	}
	if c.Provider != "bedrock" || c.BedrockRegion != "us-west-2" || len(c.InferenceModels) != 2 {
		t.Errorf("c = %+v", c)
	}

	_, err = LoadConfig("nope")
	if err == nil {
		t.Error("expected error on missing config")
	}
}

func TestFormatModel(t *testing.T) {
	cases := map[string]struct {
		in      interface{}
		wantSub string
	}{
		"string":       {"abc", "abc"},
		"with-label":   {map[string]interface{}{"name": "x", "labelOverride": "X-One"}, "X-One"},
		"with-1m":      {map[string]interface{}{"name": "y", "supports1m": true}, "[1M]"},
		"unknown-type": {42, "42"},
	}
	for n, tc := range cases {
		t.Run(n, func(t *testing.T) {
			if !strings.Contains(FormatModel(tc.in), tc.wantSub) {
				t.Errorf("FormatModel(%v) missing %q", tc.in, tc.wantSub)
			}
		})
	}
}

func TestActiveIDBadMeta(t *testing.T) {
	paths := withSandboxPaths(t)
	_ = os.MkdirAll(paths.ConfigLibrary, 0o755)
	_ = os.WriteFile(filepath.Join(paths.ConfigLibrary, "_meta.json"), []byte("not json"), 0o644)
	_, _, err := ActiveID()
	if err == nil {
		t.Error("expected parse error")
	}
}
