package claude3p_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/google-workspace-mcp-auth/internal/claude3p"
)

func TestSyncFigmaRemoteShorthand(t *testing.T) {
	policy := &claude3p.PolicyFile{
		Servers: []claude3p.ServerPolicy{
			{Figma: "remote"},
		},
	}
	result, warnings, err := claude3p.Sync(policy, "", func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(result.ConfigLibraryJSON, &cfg); err != nil {
		t.Fatal(err)
	}
	servers := cfg["managedMcpServers"].([]interface{})
	entry := servers[0].(map[string]interface{})
	if entry["name"] != "figma" {
		t.Fatalf("name: got %v", entry["name"])
	}
	if entry["url"] != "https://mcp.figma.com/mcp" {
		t.Fatalf("url: got %v", entry["url"])
	}
	if entry["oauth"] != true {
		t.Fatalf("oauth: got %v", entry["oauth"])
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "Connect in Claude Cowork") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected Connect warning, got %v", warnings)
	}
}

func TestSyncFigmaDesktopNoOAuth(t *testing.T) {
	policy := &claude3p.PolicyFile{
		Servers: []claude3p.ServerPolicy{
			{Figma: "desktop"},
		},
	}
	result, _, err := claude3p.Sync(policy, "", func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(result.ConfigLibraryJSON, &cfg); err != nil {
		t.Fatal(err)
	}
	servers := cfg["managedMcpServers"].([]interface{})
	entry := servers[0].(map[string]interface{})
	if entry["oauth"] != nil {
		t.Fatalf("desktop should not have oauth, got %v", entry["oauth"])
	}
	if entry["url"] != "http://127.0.0.1:3845/mcp" {
		t.Fatalf("url: got %v", entry["url"])
	}
}
