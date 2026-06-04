package claude3p_test

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/google-workspace-mcp-auth/internal/claude3p"
)

func TestSyncAutoModeAndBuiltinToolPolicy(t *testing.T) {
	setTestClaude3PHome(t)
	policy := &claude3p.PolicyFile{
		AutoModeEnabled: boolPtr(true),
		BuiltinToolPolicy: map[string]string{
			"WebFetch": "allow",
			"Grep":     "allow",
		},
		Egress: &claude3p.EgressPolicy{AllowedHosts: []string{"*"}},
	}
	result, _, err := claude3p.Sync(policy, "", func(string) bool { return false })
	if err != nil {
		t.Fatal(err)
	}
	var cfg map[string]interface{}
	if err := json.Unmarshal(result.ConfigLibraryJSON, &cfg); err != nil {
		t.Fatal(err)
	}
	if cfg["autoModeEnabled"] != true {
		t.Fatalf("autoModeEnabled: %v", cfg["autoModeEnabled"])
	}
	tp, ok := cfg["builtinToolPolicy"].(map[string]interface{})
	if !ok || tp["WebFetch"] != "allow" {
		t.Fatalf("builtinToolPolicy: %v", cfg["builtinToolPolicy"])
	}
	hosts := cfg["coworkEgressAllowedHosts"].([]interface{})
	if len(hosts) != 1 || hosts[0] != "*" {
		t.Fatalf("coworkEgressAllowedHosts: %v", cfg["coworkEgressAllowedHosts"])
	}
}

func boolPtr(b bool) *bool { return &b }
