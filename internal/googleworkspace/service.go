package googleworkspace

import (
	"fmt"
	"path/filepath"
	"strings"
)

// Service is a Google Workspace hosted MCP product.
type Service struct {
	ID                  string
	MCPURL              string
	DefaultScopes       string
	DefaultCallbackPort int
}

const KeychainServiceName = "google-workspace-mcp-auth"

// Catalog matches https://developers.google.com/workspace/guides/configure-mcp-servers
var Catalog = []Service{
	{
		ID:                  "gmail",
		MCPURL:              "https://gmailmcp.googleapis.com/mcp/v1",
		DefaultScopes:       "https://www.googleapis.com/auth/gmail.readonly https://www.googleapis.com/auth/gmail.compose",
		DefaultCallbackPort: 53281,
	},
	{
		ID:                  "drive",
		MCPURL:              "https://drivemcp.googleapis.com/mcp/v1",
		DefaultScopes:       "https://www.googleapis.com/auth/drive.readonly https://www.googleapis.com/auth/drive.file",
		DefaultCallbackPort: 53282,
	},
	{
		ID:                  "calendar",
		MCPURL:              "https://calendarmcp.googleapis.com/mcp/v1",
		DefaultScopes:       "https://www.googleapis.com/auth/calendar.calendarlist.readonly https://www.googleapis.com/auth/calendar.events.freebusy https://www.googleapis.com/auth/calendar.events.readonly",
		DefaultCallbackPort: 53283,
	},
	{
		ID:                  "chat",
		MCPURL:              "https://chatmcp.googleapis.com/mcp/v1",
		DefaultScopes:       "https://www.googleapis.com/auth/chat.spaces.readonly https://www.googleapis.com/auth/chat.memberships.readonly https://www.googleapis.com/auth/chat.messages.readonly https://www.googleapis.com/auth/chat.messages.create https://www.googleapis.com/auth/chat.users.readstate.readonly",
		DefaultCallbackPort: 53284,
	},
	{
		ID:                  "people",
		MCPURL:              "https://people.googleapis.com/mcp/v1",
		DefaultScopes:       "https://www.googleapis.com/auth/directory.readonly https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/contacts.readonly",
		DefaultCallbackPort: 53285,
	},
}

func Lookup(id string) (Service, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	for _, s := range Catalog {
		if s.ID == id {
			return s, nil
		}
	}
	return Service{}, fmt.Errorf("unknown service %q (valid: %s)", id, strings.Join(ServiceIDs(), ", "))
}

func ServiceIDs() []string {
	ids := make([]string, len(Catalog))
	for i, s := range Catalog {
		ids[i] = s.ID
	}
	return ids
}

func (s Service) KeychainAccount() string {
	return "google-" + s.ID
}

// ResolveFromExecutable maps bin/google-workspace-mcp-auth-gmail → gmail.
func ResolveFromExecutable(execPath string) (Service, bool) {
	base := filepath.Base(execPath)
	const prefix = "google-workspace-mcp-auth-"
	if strings.HasPrefix(base, prefix) && base != "google-workspace-mcp-auth" {
		id := strings.TrimPrefix(base, prefix)
		svc, err := Lookup(id)
		return svc, err == nil
	}
	return Service{}, false
}
