package googleworkspace

import (
	"os"
	"path/filepath"
	"testing"
)

func TestConnectorMatchesService(t *testing.T) {
	svc, _ := Lookup("gmail")
	cases := []struct {
		name string
		url  string
		want bool
	}{
		{"google-gmail", "https://gmailmcp.googleapis.com/mcp/v1", true},
		{"gmail", "", true},
		{"linear", "https://mcp.linear.app/mcp", false},
	}
	for _, tc := range cases {
		if got := connectorMatchesService(tc.name, tc.url, svc); got != tc.want {
			t.Fatalf("%q: got %v want %v", tc.name, got, tc.want)
		}
	}
}

func TestCredentialsFromClaude3P(t *testing.T) {
	root := t.TempDir()
	lib := filepath.Join(root, "configLibrary")
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_3P_HOME", root)

	cfgID := "test-config-id"
	meta := `{"activeConfigId":"test-config-id"}`
	cfg := `{
  "managedMcpServers": [{
    "name": "google-gmail",
    "url": "https://gmailmcp.googleapis.com/mcp/v1",
    "oauth": {
      "clientId": "cid.apps.googleusercontent.com",
      "clientSecret": "secret",
      "callbackPort": 53299,
      "scope": "https://www.googleapis.com/auth/gmail.readonly"
    }
  }]
}`
	if err := os.WriteFile(filepath.Join(lib, "_meta.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lib, cfgID+".json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	svc, _ := Lookup("gmail")
	c, err := credentialsFromClaude3PFull(svc)
	if err != nil {
		t.Fatal(err)
	}
	if c.ClientID != "cid.apps.googleusercontent.com" || c.ClientSecret != "secret" {
		t.Fatalf("unexpected creds: %+v", c)
	}
	if c.CallbackPort != 53299 {
		t.Fatalf("port: %d", c.CallbackPort)
	}
}

func TestOAuthHintsClientIDOnly(t *testing.T) {
	root := t.TempDir()
	lib := filepath.Join(root, "configLibrary")
	if err := os.MkdirAll(lib, 0o755); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CLAUDE_3P_HOME", root)

	cfgID := "test-config-id"
	meta := `{"activeConfigId":"test-config-id"}`
	// ally3p writes clientId as a top-level field when headersHelper is used
	// (Claude rejects entries that have both oauth and headersHelper).
	cfg := `{
  "managedMcpServers": [{
    "name": "google-gmail",
    "headersHelper": "/usr/local/bin/google-workspace-mcp-auth-gmail",
    "clientId": "cid.apps.googleusercontent.com",
    "callbackPort": 53281
  }]
}`
	if err := os.WriteFile(filepath.Join(lib, "_meta.json"), []byte(meta), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(lib, cfgID+".json"), []byte(cfg), 0o644); err != nil {
		t.Fatal(err)
	}

	svc, _ := Lookup("gmail")

	// Top-level clientId (headersHelper format) is read via readClaude3POAuthMap.
	_, oauthMap, _, err := readClaude3POAuthMap(svc)
	if err != nil {
		t.Fatal(err)
	}
	if oauthMap["clientId"] != "cid.apps.googleusercontent.com" {
		t.Fatalf("unexpected oauthMap: %+v", oauthMap)
	}

}

func TestCredentialsFromDropFile(t *testing.T) {
	root := t.TempDir()
	path := filepath.Join(root, "google-workspace-oauth.json")
	payload := `{"clientId":"a","clientSecret":"b","services":{"drive":{"callbackPort":53282}}}`
	if err := os.WriteFile(path, []byte(payload), 0o600); err != nil {
		t.Fatal(err)
	}
	t.Setenv("GOOGLE_WORKSPACE_MCP_CREDENTIALS", path)

	svc, _ := Lookup("drive")
	c, err := credentialsFromDropFile(svc)
	if err != nil {
		t.Fatal(err)
	}
	if c.ClientID != "a" || c.CallbackPort != 53282 {
		t.Fatalf("got %+v", c)
	}
}
