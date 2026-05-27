package claude3p

import "testing"

func TestMergeGmailToolPolicyExplicit(t *testing.T) {
	srv := ServerPolicy{
		GoogleService: "gmail",
		ToolPolicy: map[string]string{
			"list_drafts": "allow", "get_thread": "allow", "search_threads": "allow", "list_labels": "allow",
			"create_draft": "ask", "label_thread": "ask", "unlabel_thread": "ask", "label_message": "ask",
			"unlabel_message": "ask", "create_label": "ask", "update_label": "ask", "delete_label": "ask",
		},
	}
	tp, _, err := mergeToolPolicy(srv)
	if err != nil {
		t.Fatal(err)
	}
	if tp["list_drafts"] != "allow" || tp["create_draft"] != "ask" {
		t.Fatalf("toolPolicy: %v", tp)
	}
	if len(tp) != len(gmailToolNames()) {
		t.Fatalf("expected %d tools, got %d", len(gmailToolNames()), len(tp))
	}
}

func TestMergeGmailUnlistedDefaultsAsk(t *testing.T) {
	srv := ServerPolicy{
		GoogleService: "gmail",
		ToolPolicy:    map[string]string{"list_drafts": "allow"},
	}
	tp, _, err := mergeToolPolicy(srv)
	if err != nil {
		t.Fatal(err)
	}
	if tp["list_drafts"] != "allow" {
		t.Fatalf("list_drafts: %v", tp["list_drafts"])
	}
	if tp["create_draft"] != "ask" {
		t.Fatalf("unlisted create_draft should default ask, got %v", tp["create_draft"])
	}
}

func TestDetectMCPServerKindGoogle(t *testing.T) {
	if detectMCPServerKind(ServerPolicy{URL: "https://gmailmcp.googleapis.com/mcp/v1"}) != mcpServerGmail {
		t.Fatal("expected gmail")
	}
	if detectMCPServerKind(ServerPolicy{URL: "https://mcp.slack.com/mcp"}) != mcpServerSlack {
		t.Fatal("expected slack")
	}
}
