package claude3p_test

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/google-workspace-mcp-auth/internal/claude3p"
)

func TestSyncPostHogPartialToolPolicyDefaultsAsk(t *testing.T) {
	if testing.Short() {
		t.Skip("network")
	}
	setTestClaude3PHome(t)
	policy := &claude3p.PolicyFile{
		Servers: []claude3p.ServerPolicy{
			{
				Name:  "posthog",
				URL:   "https://mcp.posthog.com/mcp?features=flags",
				OAuth: true,
				ToolPolicy: map[string]string{
					"feature-flag-get-all": "allow",
					"create-feature-flag":  "ask",
				},
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
	tp, ok := entry["toolPolicy"].(map[string]interface{})
	if !ok || len(tp) == 0 {
		t.Fatal("expected toolPolicy")
	}
	if tp["feature-flag-get-all"] != "allow" {
		t.Fatalf("feature-flag-get-all: got %v", tp["feature-flag-get-all"])
	}
	if tp["create-feature-flag"] != "ask" {
		t.Fatalf("create-feature-flag: got %v", tp["create-feature-flag"])
	}
	if tp["feature-flag-get-definition"] != "ask" {
		t.Fatalf("unlisted tool should default ask, got %v", tp["feature-flag-get-definition"])
	}
}
