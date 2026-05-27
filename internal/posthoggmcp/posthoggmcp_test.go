package posthoggmcp_test

import (
	"testing"

	"github.com/anthropics/google-workspace-mcp-auth/internal/posthoggmcp"
)

func TestIsRemote(t *testing.T) {
	if !posthoggmcp.IsRemote("https://mcp.posthog.com/mcp") {
		t.Fatal("expected remote")
	}
	if posthoggmcp.IsRemote("https://mcp.figma.com/mcp") {
		t.Fatal("figma should not match posthog")
	}
}
