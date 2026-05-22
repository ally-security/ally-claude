package version

import "runtime"

// Repo is the GitHub repository in <owner>/<name> form used by self-update.
const Repo = "ally-security/ally-claude"

var (
	Version = "dev"
	Commit  = "none"
)

func String() string {
	return Version + " (" + Commit + ") " + runtime.Version()
}
