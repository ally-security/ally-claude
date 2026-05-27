package figmamcp_test

import (
	"testing"

	"github.com/anthropics/google-workspace-mcp-auth/internal/figmamcp"
)

func TestIsRemote(t *testing.T) {
	if !figmamcp.IsRemote("https://mcp.figma.com/mcp") {
		t.Fatal("expected remote")
	}
	if figmamcp.IsRemote("http://127.0.0.1:3845/mcp") {
		t.Fatal("desktop should not be remote")
	}
}

func TestIsDesktop(t *testing.T) {
	if !figmamcp.IsDesktop("http://127.0.0.1:3845/mcp") {
		t.Fatal("expected desktop")
	}
	if !figmamcp.IsDesktop("http://localhost:3845/mcp") {
		t.Fatal("expected localhost desktop")
	}
}
