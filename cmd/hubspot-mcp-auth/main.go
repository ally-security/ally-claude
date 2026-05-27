// hubspot-mcp-auth — headersHelper for HubSpot hosted MCP (bypasses Claude OAuth).
package main

import (
	"fmt"
	"os"

	"github.com/anthropics/google-workspace-mcp-auth/internal/hubspotmcp"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		return cmdHeaders()
	}
	switch args[0] {
	case "headers", "header", "print", "":
		return cmdHeaders()
	case "login", "auth":
		return cmdLogin()
	case "help", "-h", "--help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		usage()
		return 2
	}
}

func cmdHeaders() int {
	if err := hubspotmcp.PrintHeadersJSON(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func cmdLogin() int {
	if err := hubspotmcp.Login(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func usage() {
	fmt.Fprint(os.Stderr, `hubspot-mcp-auth — HubSpot hosted MCP headersHelper

Usage:
  hubspot-mcp-auth              Print Authorization header JSON (for Claude headersHelper)
  hubspot-mcp-auth login        Run HubSpot OAuth and save user token to Keychain

Credentials are installed by: ally3p claude sync <policy.yaml>
`)
}
