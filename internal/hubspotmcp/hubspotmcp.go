// Package hubspotmcp holds constants for HubSpot's hosted MCP server.
package hubspotmcp

import "strings"

const (
	RemoteMCPURL        = "https://mcp.hubspot.com/anthropic"
	DefaultCallbackPort = 3119
	KeychainService     = "hubspot-mcp-auth"
)

const (
	ClientKeychainAccount = "oauth-client"
	UserTokenAccount      = "user-token"
)

// IsRemote reports whether url points at HubSpot's hosted MCP server.
func IsRemote(url string) bool {
	u := strings.ToLower(strings.TrimSpace(url))
	return strings.Contains(u, "mcp.hubspot.com")
}
