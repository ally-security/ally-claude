package googleworkspace

import "testing"

func TestParseInvocationExplicitService(t *testing.T) {
	inv, _, err := ParseInvocation("/usr/local/bin/google-workspace-mcp-auth", []string{"drive", "login"})
	if err != nil {
		t.Fatal(err)
	}
	if inv.Service.ID != "drive" || inv.Command != "login" {
		t.Fatalf("got %+v", inv)
	}
}

func TestParseInvocationFromSymlink(t *testing.T) {
	inv, _, err := ParseInvocation("/opt/bin/google-workspace-mcp-auth-gmail", []string{"verify"})
	if err != nil {
		t.Fatal(err)
	}
	if inv.Service.ID != "gmail" || inv.Command != "verify" {
		t.Fatalf("got %+v", inv)
	}
}

func TestParseInvocationGlobalList(t *testing.T) {
	inv, _, err := ParseInvocation("google-workspace-mcp-auth", []string{"list"})
	if err != nil {
		t.Fatal(err)
	}
	if inv.Command != "list" {
		t.Fatalf("got %+v", inv)
	}
}

func TestResolveFromExecutable(t *testing.T) {
	svc, ok := ResolveFromExecutable("google-workspace-mcp-auth-calendar")
	if !ok || svc.ID != "calendar" {
		t.Fatalf("calendar: ok=%v svc=%+v", ok, svc)
	}
}
