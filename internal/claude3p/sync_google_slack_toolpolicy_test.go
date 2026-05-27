package claude3p_test

import (
	"encoding/json"
	"testing"

	"github.com/anthropics/google-workspace-mcp-auth/internal/claude3p"
)

func gmailToolPolicy() map[string]string {
	return map[string]string{
		"list_drafts": "allow", "get_thread": "allow", "search_threads": "allow", "list_labels": "allow",
		"create_draft": "ask", "label_thread": "ask", "unlabel_thread": "ask", "label_message": "ask",
		"unlabel_message": "ask", "create_label": "ask", "update_label": "ask", "delete_label": "ask",
	}
}

func slackToolPolicy() map[string]string {
	return map[string]string{
		"slack_search_public": "allow", "slack_search_public_and_private": "allow",
		"slack_search_channels": "allow", "slack_search_users": "allow",
		"slack_read_channel": "allow", "slack_read_thread": "allow", "slack_read_canvas": "allow",
		"slack_read_user_profile": "allow", "slack_list_channel_members": "allow",
		"slack_read_file": "allow", "slack_search_emojis": "allow", "slack_get_reactions": "allow",
		"slack_schedule_message": "ask", "slack_add_reaction": "ask",
		"slack_create_conversation": "ask", "slack_create_canvas": "ask",
		"slack_update_canvas": "ask", "slack_send_message_draft": "ask",
	}
}

func TestSyncGoogleGmailToolPolicy(t *testing.T) {
	setTestClaude3PHome(t)
	policy := &claude3p.PolicyFile{
		Servers: []claude3p.ServerPolicy{
			{
				Name:          "google-gmail",
				GoogleService: "gmail",
				ClientID:      "test.apps.googleusercontent.com",
				ToolPolicy:    gmailToolPolicy(),
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
	entry := findServer(cfg, "google-gmail")
	tp := entry["toolPolicy"].(map[string]interface{})
	if tp["list_drafts"] != "allow" || tp["create_draft"] != "ask" {
		t.Fatalf("toolPolicy: %v", tp)
	}
}

func TestSyncSlackToolPolicy(t *testing.T) {
	setTestClaude3PHome(t)
	policy := &claude3p.PolicyFile{
		Servers: []claude3p.ServerPolicy{
			{
				Name:       "slack",
				URL:        "https://mcp.slack.com/mcp",
				ToolPolicy: slackToolPolicy(),
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
	entry := findServer(cfg, "slack")
	tp := entry["toolPolicy"].(map[string]interface{})
	if tp["slack_read_channel"] != "allow" || tp["slack_send_message_draft"] != "ask" {
		t.Fatalf("toolPolicy: %v", tp)
	}
}

func findServer(cfg map[string]interface{}, name string) map[string]interface{} {
	for _, s := range cfg["managedMcpServers"].([]interface{}) {
		e := s.(map[string]interface{})
		if e["name"] == name {
			return e
		}
	}
	return nil
}
