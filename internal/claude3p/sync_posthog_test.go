package claude3p_test

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/google-workspace-mcp-auth/internal/claude3p"
)

func TestSyncPostHog(t *testing.T) {
	setTestClaude3PHome(t)
	policy := &claude3p.PolicyFile{
		Servers: []claude3p.ServerPolicy{
			{
				Name:  "posthog",
				URL:   "https://mcp.posthog.com/mcp",
				OAuth: true,
			},
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
	if entry["name"] != "posthog" {
		t.Fatalf("name: got %v", entry["name"])
	}
	if entry["url"] != "https://mcp.posthog.com/mcp" {
		t.Fatalf("url: got %v", entry["url"])
	}
	if entry["oauth"] != true {
		t.Fatalf("oauth: got %v", entry["oauth"])
	}
}
