package claude3p

import (
	"fmt"
	"sort"
	"strings"
)

type mcpServerKind int

const (
	mcpServerUnknown mcpServerKind = iota
	mcpServerPostHog
	mcpServerGmail
	mcpServerDrive
	mcpServerCalendar
	mcpServerSlack
)

func detectMCPServerKind(srv ServerPolicy) mcpServerKind {
	url := strings.ToLower(strings.TrimSpace(srv.URL))
	name := strings.ToLower(strings.TrimSpace(srv.Name))
	switch {
	case isPostHogMCP(url):
		return mcpServerPostHog
	case strings.Contains(url, "gmailmcp.googleapis.com") || name == "google-gmail" || name == "gmail" || srv.GoogleService == "gmail":
		return mcpServerGmail
	case strings.Contains(url, "drivemcp.googleapis.com") || name == "google-drive" || name == "drive" || srv.GoogleService == "drive":
		return mcpServerDrive
	case strings.Contains(url, "calendarmcp.googleapis.com") || name == "google-calendar" || name == "calendar" || srv.GoogleService == "calendar":
		return mcpServerCalendar
	case strings.Contains(url, "mcp.slack.com") || name == "slack":
		return mcpServerSlack
	default:
		return mcpServerUnknown
	}
}

// knownToolNamesForServer returns every tool name sync can enumerate for this MCP server.
func knownToolNamesForServer(srv ServerPolicy) ([]string, error) {
	switch detectMCPServerKind(srv) {
	case mcpServerGmail:
		return gmailToolNames(), nil
	case mcpServerDrive:
		return driveToolNames(), nil
	case mcpServerCalendar:
		return calendarToolNames(), nil
	case mcpServerSlack:
		return slackToolNames(), nil
	case mcpServerPostHog:
		return posthogToolNames(srv.URL)
	default:
		return nil, nil
	}
}

func gmailToolNames() []string {
	return []string{
		"list_drafts", "get_thread", "search_threads", "list_labels",
		"create_draft", "label_thread", "unlabel_thread", "label_message",
		"unlabel_message", "create_label", "update_label", "delete_label",
	}
}

func driveToolNames() []string {
	return []string{
		"download_file_content", "get_file_metadata", "get_file_permissions",
		"list_recent_files", "read_file_content", "search_files",
		"copy_file", "create_file",
	}
}

func calendarToolNames() []string {
	return []string{
		"list_events", "get_event", "list_calendars",
		"suggest_time", "create_event", "update_event", "delete_event", "respond_to_event",
	}
}

func slackToolNames() []string {
	return []string{
		"slack_search_public", "slack_search_public_and_private",
		"slack_search_channels", "slack_search_users",
		"slack_read_channel", "slack_read_thread", "slack_read_canvas",
		"slack_read_user_profile", "slack_list_channel_members",
		"slack_read_file", "slack_search_emojis", "slack_get_reactions",
		"slack_schedule_message", "slack_add_reaction",
		"slack_create_conversation", "slack_create_canvas",
		"slack_update_canvas", "slack_send_message_draft",
	}
}

func posthogToolNames(mcpURL string) ([]string, error) {
	features, err := posthogParseFeaturesParam(mcpURL)
	if err != nil {
		return nil, err
	}
	defs, err := posthogFetchToolDefinitions()
	if err != nil {
		return nil, err
	}
	var names []string
	for name, def := range defs {
		if len(features) > 0 && !posthogFeatureMatches(def.Feature, features) {
			continue
		}
		names = append(names, name)
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no PostHog tools matched (features filter may be too narrow)")
	}
	sort.Strings(names)
	return names, nil
}

func countToolPolicyActions(m map[string]string) (allow, ask, blocked int) {
	for _, a := range m {
		switch a {
		case "allow":
			allow++
		case "ask":
			ask++
		case "blocked":
			blocked++
		}
	}
	return allow, ask, blocked
}
