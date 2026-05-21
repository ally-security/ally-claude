package version

import (
	"runtime"
	"strings"
	"testing"
)

func TestStringIncludesAllParts(t *testing.T) {
	prevV, prevC := Version, Commit
	defer func() { Version, Commit = prevV, prevC }()
	Version = "1.2.3"
	Commit = "abc123"

	got := String()
	for _, want := range []string{"1.2.3", "abc123", runtime.Version()} {
		if !strings.Contains(got, want) {
			t.Errorf("String() = %q, missing %q", got, want)
		}
	}
}

func TestRepoConst(t *testing.T) {
	if Repo == "" {
		t.Fatal("Repo must not be empty")
	}
	if !strings.Contains(Repo, "/") {
		t.Errorf("Repo = %q, expected owner/name form", Repo)
	}
}
