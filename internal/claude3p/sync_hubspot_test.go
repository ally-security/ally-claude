package claude3p_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/google-workspace-mcp-auth/internal/claude3p"
)

func TestSyncHubSpotShorthand(t *testing.T) {
	setTestClaude3PHome(t)
	helperDir := t.TempDir()
	helper := filepath.Join(helperDir, "hubspot-mcp-auth")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	policy := &claude3p.PolicyFile{
		Servers: []claude3p.ServerPolicy{
			{
				HubSpot: true,
				OAuth: map[string]interface{}{
					"client_id":     "test-client",
					"client_secret": "test-secret",
				},
			},
		},
	}
	result, _, err := claude3p.Sync(policy, helperDir, func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(result.ConfigLibraryJSON, &cfg); err != nil {
		t.Fatal(err)
	}
	servers := cfg["managedMcpServers"].([]interface{})
	entry := servers[0].(map[string]interface{})
	if entry["name"] != "hubspot" {
		t.Fatalf("name: got %v", entry["name"])
	}
	if entry["url"] != "https://mcp.hubspot.com/anthropic" {
		t.Fatalf("url: got %v", entry["url"])
	}
	if entry["headersHelper"] == nil {
		t.Fatalf("expected headersHelper, got %v", entry)
	}
	if entry["oauth"] != nil {
		t.Fatalf("hubspot should not use oauth block when helper resolves, got %v", entry["oauth"])
	}
}
