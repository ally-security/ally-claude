// Package posthoggmcp holds constants for PostHog's hosted MCP server.
package posthoggmcp

import "strings"

const RemoteMCPURL = "https://mcp.posthog.com/mcp"

// IsRemote reports whether url points at PostHog's hosted MCP server.
func IsRemote(url string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(url)), "mcp.posthog.com")
}
