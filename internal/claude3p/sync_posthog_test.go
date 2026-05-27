package claude3p_test

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/anthropics/google-workspace-mcp-auth/internal/claude3p"
)

func TestSyncPostHogShorthand(t *testing.T) {
	setTestClaude3PHome(t)
	policy := &claude3p.PolicyFile{
		Servers: []claude3p.ServerPolicy{
			{PostHog: true},
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
	if entry["name"] != "posthog" {
		t.Fatalf("name: got %v", entry["name"])
	}
	if entry["url"] != "https://mcp.posthog.com/mcp" {
		t.Fatalf("url: got %v", entry["url"])
	}
	if entry["oauth"] != true {
		t.Fatalf("oauth: got %v", entry["oauth"])
	}
	found := false
	for _, w := range warnings {
		if strings.Contains(w, "PostHog MCP uses oauth:true") {
			found = true
			break
		}
	}
	if !found {
		t.Fatalf("expected PostHog oauth warning, got %v", warnings)
	}
}
