package claude3p_test

import (
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/anthropics/google-workspace-mcp-auth/internal/claude3p"
)

func TestSyncSlackToolPolicyWithHeadersHelper(t *testing.T) {
	setTestClaude3PHome(t)
	helperDir := t.TempDir()
	helper := filepath.Join(helperDir, "slack-mcp-auth")
	if err := os.WriteFile(helper, []byte("#!/bin/sh\n"), 0o755); err != nil {
		t.Fatal(err)
	}

	policy := &claude3p.PolicyFile{
		Servers: []claude3p.ServerPolicy{
			{
				Name: "slack",
				URL:  "https://mcp.slack.com/mcp",
				OAuth: map[string]interface{}{
					"client_id":     "x",
					"client_secret": "y",
				},
				ToolPolicy: map[string]string{
					"slack_read_channel": "allow",
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
	entry := cfg["managedMcpServers"].([]interface{})[0].(map[string]interface{})
	tp, ok := entry["toolPolicy"].(map[string]interface{})
	if !ok {
		t.Fatalf("expected toolPolicy on slack entry, got %v", entry)
	}
	if tp["slack_read_channel"] != "allow" {
		t.Fatalf("explicit policy missing: %v", tp)
	}
}

func TestIsReadOnlyToolName(t *testing.T) {
	cases := map[string]bool{
		"get_event": true, "list_events": true, "search_files": true,
		"create_event": false, "delete_label": false, "slack_send_message": false,
		"query_crm_data": true, "submit_feedback": true,
	}
	for name, want := range cases {
		if claude3p.IsReadOnlyToolName(name) != want {
			t.Fatalf("%s readonly=%v want %v", name, !want, want)
		}
	}
}
