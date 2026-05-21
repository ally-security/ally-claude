package version

import "runtime"

var (
	Version = "dev"
	Commit  = "none"
)

func String() string {
	return Version + " (" + Commit + ") " + runtime.Version()
}
