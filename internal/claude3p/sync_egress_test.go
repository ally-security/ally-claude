package claude3p_test

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/google-workspace-mcp-auth/internal/claude3p"
)

func TestSyncEgressAllowedHosts(t *testing.T) {
	setTestClaude3PHome(t)
	policy := &claude3p.PolicyFile{
		Egress: &claude3p.EgressPolicy{
			AllowedHosts: []string{"ally.security", " docs.ally.security ", ""},
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
	raw, ok := cfg["coworkEgressAllowedHosts"]
	if !ok {
		t.Fatalf("coworkEgressAllowedHosts missing from config: %s", result.ConfigLibraryJSON)
	}
	hosts := raw.([]interface{})
	if len(hosts) != 2 {
		t.Fatalf("expected 2 trimmed hosts, got %v", hosts)
	}
	if hosts[0] != "ally.security" || hosts[1] != "docs.ally.security" {
		t.Fatalf("unexpected hosts (should be trimmed, blanks dropped): %v", hosts)
	}
}

func TestSyncEgressDefaultsToUnrestricted(t *testing.T) {
	setTestClaude3PHome(t)
	for name, policy := range map[string]*claude3p.PolicyFile{
		"nil egress":     {},
		"empty hosts":    {Egress: &claude3p.EgressPolicy{AllowedHosts: []string{"  ", ""}}},
		"empty egress":   {Egress: &claude3p.EgressPolicy{}},
	} {
		t.Run(name, func(t *testing.T) {
			result, _, err := claude3p.Sync(policy, "", func(string) bool { return false })
			if err != nil {
				t.Fatal(err)
			}
			var cfg map[string]interface{}
			if err := json.Unmarshal(result.ConfigLibraryJSON, &cfg); err != nil {
				t.Fatal(err)
			}
			hosts := cfg["coworkEgressAllowedHosts"].([]interface{})
			if len(hosts) != 1 || hosts[0] != "*" {
				t.Fatalf("expected [\"*\"], got %v", hosts)
			}
		})
	}
}
