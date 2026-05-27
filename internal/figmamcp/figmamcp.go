// Package figmamcp holds constants for Figma hosted and desktop MCP servers.
package figmamcp

import "strings"

const (
	RemoteMCPURL  = "https://mcp.figma.com/mcp"
	DesktopMCPURL = "http://127.0.0.1:3845/mcp"
)

// IsRemote reports whether url points at Figma's hosted MCP server.
func IsRemote(url string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(url)), "mcp.figma.com")
}

// IsDesktop reports whether url points at the Figma desktop app's local MCP server.
func IsDesktop(url string) bool {
	u := strings.ToLower(strings.TrimSpace(url))
	return strings.Contains(u, "127.0.0.1:3845") || strings.Contains(u, "localhost:3845")
}
