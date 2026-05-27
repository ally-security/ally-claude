// slack-mcp-auth — headersHelper for Slack hosted MCP (bypasses Claude OAuth).
package main

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/anthropics/google-workspace-mcp-auth/internal/slackmcp"
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
	if err := slackmcp.PrintHeadersJSON(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func cmdLogin() int {
	if err := slackmcp.Login(); err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}
	return 0
}

func usage() {
	fmt.Fprint(os.Stderr, `slack-mcp-auth — Slack hosted MCP headersHelper

Usage:
  slack-mcp-auth              Print Authorization header JSON (for Claude headersHelper)
  slack-mcp-auth login        Run Slack OAuth and save user token to Keychain

Credentials are installed by: ally3p claude sync <policy.yaml>
`)
}

func parseInstallCredentials(args []string) (clientID, clientSecret string, port int, ok bool) {
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
		case "--callback-port":
			i++
			if i < len(args) {
				port, _ = strconv.Atoi(args[i])
			}
		}
	}
	return clientID, clientSecret, port, strings.TrimSpace(clientID) != "" && strings.TrimSpace(clientSecret) != ""
}
