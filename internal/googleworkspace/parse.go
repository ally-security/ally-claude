package googleworkspace

import (
	"fmt"
	"os"
	"strings"
)

var commands = map[string]struct{}{
	"headers": {}, "header": {}, "print": {},
	"login": {}, "auth": {},
	"verify":              {},
	"logout":              {},
	"list":                {},
	"install-credentials": {},
	"help":                {}, "-h": {}, "--help": {},
}

var globalCommands = map[string]struct{}{
	"list": {}, "install-credentials": {},
	"help": {}, "-h": {}, "--help": {},
}

// Invocation holds parsed CLI state.
type Invocation struct {
	Service Service
	Command string
}

// ParseInvocation resolves service + subcommand from argv0, env, and args.
// The returned rest slice is arguments after the subcommand name.
func ParseInvocation(execPath string, args []string) (Invocation, []string, error) {
	if len(args) > 0 {
		cmd := strings.ToLower(args[0])
		if _, ok := globalCommands[cmd]; ok {
			return Invocation{Command: cmd}, args[1:], nil
		}
	}

	var svc Service
	var ok bool

	if s, found := ResolveFromExecutable(execPath); found {
		svc = s
		ok = true
	}

	rest := args
	if len(rest) > 0 {
		if _, isCmd := commands[strings.ToLower(rest[0])]; !isCmd {
			if s, err := Lookup(rest[0]); err == nil {
				svc = s
				ok = true
				rest = rest[1:]
			}
		}
	}

	if !ok {
		if id := os.Getenv("GOOGLE_MCP_SERVICE"); id != "" {
			s, err := Lookup(id)
			if err != nil {
				return Invocation{}, nil, err
			}
			svc = s
			ok = true
		}
	}

	if !ok {
		return Invocation{}, nil, fmt.Errorf("service required (e.g. gmail, drive); use %s-<service> wrapper", "google-workspace-mcp-auth")
	}

	cmd := "headers"
	if len(rest) > 0 {
		cmd = strings.ToLower(rest[0])
		rest = rest[1:]
	}

	return Invocation{Service: svc, Command: cmd}, rest, nil
}
