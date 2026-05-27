// google-workspace-mcp-auth — OAuth + headersHelper for Google Workspace hosted MCP servers.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/google-workspace-mcp-auth/internal/googleworkspace"
)

func main() {
	googleworkspace.SetStderr(os.Stderr)
	os.Exit(run(os.Args[0], os.Args[1:]))
}

func run(execPath string, args []string) int {
	inv, rest, err := googleworkspace.ParseInvocation(execPath, args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		usage()
		return 2
	}

	switch inv.Command {
	case "list":
		return cmdList()
	case "install-credentials":
		return cmdInstallCredentials(rest)
	case "help", "-h", "--help":
		usage()
		return 0
	}

	store := googleworkspace.NewStore(inv.Service)

	switch inv.Command {
	case "headers", "header", "print", "":
		return cmdHeaders(inv.Service, store)
	case "login", "auth":
		return cmdLogin(inv.Service, store)
	case "verify":
		return cmdVerify(inv.Service, store)
	case "logout":
		return cmdLogout(inv.Service, store)
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", inv.Command)
		usage()
		return 2
	}
}

func cmdHeaders(svc googleworkspace.Service, store googleworkspace.Store) int {
	if err := googleworkspace.PrintHeadersJSON(store, svc); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func cmdLogin(svc googleworkspace.Service, _ googleworkspace.Store) int {
	if err := googleworkspace.EnsureUnifiedLogin(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "Register redirect URI: http://127.0.0.1:%d/callback\n", googleworkspace.UnifiedCallbackPort)
	return 0
}

func cmdVerify(svc googleworkspace.Service, store googleworkspace.Store) int {
	cfg, err := googleworkspace.LoadConfig(svc)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	fmt.Fprintf(os.Stderr, "== [%s] MCP endpoint %s ==\n", svc.ID, svc.MCPURL)
	if err := googleworkspace.VerifyMCPEndpoint(svc); err != nil {
		fmt.Fprintf(os.Stderr, "  FAIL: %v\n", err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "  OK")

	fmt.Fprintln(os.Stderr, "== OAuth redirect URI ==")
	if err := googleworkspace.VerifyRedirectURI(cfg); err != nil {
		fmt.Fprintf(os.Stderr, "  FAIL: %v\n", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "  OK (%s)\n", cfg.RedirectURI())

	fmt.Fprintln(os.Stderr, "== Keychain headers ==")
	if err := googleworkspace.PrintHeadersJSON(store, svc); err != nil {
		fmt.Fprintf(os.Stderr, "  FAIL: %v\n", err)
		return 1
	}
	fmt.Fprintln(os.Stderr, "  OK (Bearer header on stdout)")
	return 0
}

func cmdLogout(svc googleworkspace.Service, store googleworkspace.Store) int {
	_ = svc
	if err := store.Delete(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "[%s] removed keychain entry %s / %s\n", svc.ID, store.Service, store.Account)
	return 0
}

func cmdList() int {
	for _, s := range googleworkspace.Catalog {
		fmt.Printf("%s\t%s\tport %d\n", s.ID, s.MCPURL, s.DefaultCallbackPort)
	}
	return 0
}

func cmdInstallCredentials(args []string) int {
	var clientID, clientSecret, path string
	useFile := false
	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--client-id":
			i++
			if i < len(args) {
				clientID = args[i]
			}
		case "--client-secret":
			i++
			if i < len(args) {
				clientSecret = args[i]
			}
		case "--path":
			i++
			if i < len(args) {
				path = args[i]
			}
		case "--file":
			useFile = true
		}
	}
	if clientID == "" || clientSecret == "" {
		fmt.Fprintln(os.Stderr, "usage: install-credentials --client-id ID --client-secret SECRET [--file] [--path FILE]")
		return 2
	}
	if useFile || path != "" {
		if err := googleworkspace.WriteCredentialsDropFile(path, clientID, clientSecret); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if path == "" {
			path = googleworkspace.DefaultCredentialsPath()
		}
		fmt.Fprintf(os.Stderr, "Wrote enterprise credentials to %s\n", path)
		return 0
	}
	if err := googleworkspace.SaveClientCredentials(clientID, clientSecret); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "Saved OAuth client to %s (once per Mac)\n", googleworkspace.ClientCredentialsKeychainLocation())
	return 0
}

func usage() {
	fmt.Fprintf(os.Stderr, `google-workspace-mcp-auth — Google Workspace hosted MCP (OAuth + headersHelper)

Usage:
  google-workspace-mcp-auth <service> [command]
  google-workspace-mcp-auth-<service> [command]

Commands (default: headers):
  headers              Print Authorization header JSON (auto sign-in if needed)
  login                Browser OAuth → keychain (optional; headers does this too)
  verify               Check MCP URL, redirect URI, headers
  logout               Delete keychain entry for this service
  list                 List supported services
  install-credentials  IT: save OAuth client to Keychain (once per Mac)
  install-credentials --file  IT: write oauth JSON file instead (Linux / legacy)

End users: no environment variables. IT runs install-credentials once per Mac;
Claude configLibrary only needs oauth.clientId + callbackPort (no secret on disk).

Services: %s

`,
		strings.Join(googleworkspace.ServiceIDs(), ", "),
	)
}
