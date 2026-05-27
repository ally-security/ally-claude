// ally3p — manage Claude Cowork 3P configuration from a YAML policy file.
package main

import (
	"fmt"
	"os"
	"strings"

	"github.com/anthropics/google-workspace-mcp-auth/internal/claude3p"
	"github.com/anthropics/google-workspace-mcp-auth/internal/figmamcp"
	"github.com/anthropics/google-workspace-mcp-auth/internal/googleworkspace"
	"github.com/anthropics/google-workspace-mcp-auth/internal/hubspotmcp"
	"github.com/anthropics/google-workspace-mcp-auth/internal/posthoggmcp"
	"github.com/anthropics/google-workspace-mcp-auth/internal/slackmcp"
)

func main() {
	os.Exit(run(os.Args[1:]))
}

func run(args []string) int {
	if len(args) == 0 {
		usage()
		return 0
	}
	switch args[0] {
	case "claude":
		return cmdClaude(args[1:])
	case "prereq":
		return cmdPrereq(args[1:])
	case "help", "-h", "--help":
		usage()
		return 0
	default:
		fmt.Fprintf(os.Stderr, "unknown command %q\n\n", args[0])
		usage()
		return 2
	}
}

// ── claude sync ────────────────────────────────────────────────────────────

func cmdClaude(args []string) int {
	if len(args) == 0 {
		printClaudeUsage()
		return 2
	}
	switch args[0] {
	case "sync":
		return cmdClaudeSync(args[1:])
	case "login":
		return cmdClaudeLogin(args[1:])
	default:
		fmt.Fprintf(os.Stderr, "unknown claude subcommand %q\n", args[0])
		return 2
	}
}

func cmdClaudeLogin(args []string) int {
	service, policyPath, err := parseLoginArgs(args)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 2
	}

	var policy *claude3p.PolicyFile
	if policyPath != "" {
		policy, err = claude3p.ParsePolicyFile(policyPath)
		if err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
	}

	switch service {
	case "":
		if policy == nil {
			printLoginUsage()
			return 2
		}
		service = "all"
		fallthrough
	case "all":
		if policy == nil {
			fmt.Fprintln(os.Stderr, "login all requires a policy.yaml")
			printLoginUsage()
			return 2
		}
		if err := installSlackCredsFromPolicy(policy); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if err := installHubSpotCredsFromPolicy(policy); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		if policyHasSlack(policy) {
			if err := slackmcp.Login(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		}
		if policyHasHubSpot(policy) {
			if err := hubspotmcp.EnsureLogin(); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		}
		if policyHasGoogle(policy) {
			if err := googleworkspace.ForceUnifiedLogin(policyGoogleScopes(policy)); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		}
		return 0

	case "slack":
		if policy != nil {
			if err := installSlackCredsFromPolicy(policy); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		}
		if err := slackmcp.Login(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0

	case "hubspot":
		if policy != nil {
			if err := installHubSpotCredsFromPolicy(policy); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		}
		if err := hubspotmcp.EnsureLogin(); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0

	case "figma":
		fmt.Println("Figma remote MCP uses Claude Cowork OAuth (oauth: true, dynamic client registration).")
		fmt.Println("No external login binary — sync your policy, restart Claude, then click Connect for the figma server.")
		if policy != nil {
			for _, s := range policy.Servers {
				if figmamcp.IsRemote(s.URL) || figmamcp.IsDesktop(s.URL) {
					fmt.Printf("  Policy entry: %q → %s\n", s.Name, s.URL)
				}
			}
		}
		return 0

	case "posthog":
		fmt.Println("PostHog MCP uses Claude Cowork OAuth (oauth: true, dynamic client registration).")
		fmt.Println("No external login binary — sync your policy, restart Claude, then click Connect for the posthog server.")
		if policy != nil {
			for _, s := range policy.Servers {
				if posthoggmcp.IsRemote(s.URL) {
					fmt.Printf("  Policy entry: %q → %s\n", s.Name, s.URL)
				}
			}
		}
		return 0

	case "google":
		var scopes string
		if policy != nil {
			scopes = policyGoogleScopes(policy)
		}
		if scopes == "" {
			var err error
			scopes, err = googleworkspace.MergedScopesFromClaude3P()
			if err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
		}
		if err := googleworkspace.ForceUnifiedLogin(scopes); err != nil {
			fmt.Fprintln(os.Stderr, err)
			return 1
		}
		return 0

	default:
		fmt.Fprintf(os.Stderr, "unknown login target %q\n\n", service)
		printLoginUsage()
		return 2
	}
}

func parseLoginArgs(args []string) (service, policyPath string, err error) {
	for _, a := range args {
		if strings.HasPrefix(a, "-") {
			return "", "", fmt.Errorf("unknown flag %q", a)
		}
		if isPolicyPath(a) {
			policyPath = a
			continue
		}
		if service != "" {
			return "", "", fmt.Errorf("unexpected argument %q", a)
		}
		service = strings.ToLower(strings.TrimSpace(a))
	}
	return service, policyPath, nil
}

func isPolicyPath(s string) bool {
	lower := strings.ToLower(s)
	return strings.HasSuffix(lower, ".yaml") || strings.HasSuffix(lower, ".yml")
}

func installSlackCredsFromPolicy(policy *claude3p.PolicyFile) error {
	for _, s := range policy.Servers {
		if !isSlackServer(s) {
			continue
		}
		clientID, clientSecret := slackClientFromPolicy(s)
		if clientID == "" || clientSecret == "" {
			continue
		}
		port := s.CallbackPort
		if port == 0 {
			port = slackmcp.DefaultCallbackPort
		}
		return slackmcp.SaveClientCredentials(clientID, clientSecret, port)
	}
	return nil
}

func installHubSpotCredsFromPolicy(policy *claude3p.PolicyFile) error {
	for _, s := range policy.Servers {
		if !isHubSpotServer(s) {
			continue
		}
		clientID, clientSecret := hubspotClientFromPolicy(s)
		if clientID == "" || clientSecret == "" {
			continue
		}
		port := s.CallbackPort
		if port == 0 {
			port = hubspotmcp.DefaultCallbackPort
		}
		return hubspotmcp.SaveClientCredentials(clientID, clientSecret, port)
	}
	return nil
}

func cmdClaudeSync(args []string) int {
	var policyPath string
	var dry bool
	var helperDir string

	for i := 0; i < len(args); i++ {
		switch args[i] {
		case "--dry", "--dry-run":
			dry = true
		case "--helper-dir":
			i++
			if i < len(args) {
				helperDir = args[i]
			}
		default:
			if !strings.HasPrefix(args[i], "-") && policyPath == "" {
				policyPath = args[i]
			} else {
				fmt.Fprintf(os.Stderr, "unknown flag %q\n", args[i])
				return 2
			}
		}
	}

	if policyPath == "" {
		fmt.Fprintln(os.Stderr, "usage: ally3p claude sync <policy.yaml> [--dry]")
		return 2
	}
	if !dry && needsPrereqs(helperDir) {
		fmt.Fprintln(os.Stderr, "missing google helper wrappers; running ally3p prereq ...")
		if err := ensurePrereqs(helperDir); err != nil {
			fmt.Fprintf(os.Stderr, "prereq failed: %v\n", err)
			return 1
		}
	}

	policy, err := claude3p.ParsePolicyFile(policyPath)
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	// If the YAML includes Google client secrets, install them into Keychain first
	// (so the generated Claude config never contains clientSecret on disk).
	if !dry {
		for _, s := range policy.Servers {
			if strings.TrimSpace(s.GoogleService) == "" {
				continue
			}
			if strings.TrimSpace(s.ClientSecret) == "" {
				continue
			}
			serviceID := strings.TrimSpace(s.GoogleService)
			serviceID = strings.TrimPrefix(serviceID, "google-")
			clientID := strings.TrimSpace(s.ClientID)
			if clientID == "" {
				fmt.Fprintf(os.Stderr, "google_service %q requires client_id\n", serviceID)
				return 2
			}
			if err := googleworkspace.SaveClientCredentialsForService(serviceID, clientID, strings.TrimSpace(s.ClientSecret)); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Fprintf(os.Stderr, "✓ installed %s client secret into Keychain (oauth-client-%s)\n", serviceID, serviceID)
		}
		for _, s := range policy.Servers {
			if !isSlackServer(s) {
				continue
			}
			if err := installSlackCredsFromPolicy(policy); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Fprintln(os.Stderr, "✓ installed Slack OAuth client into Keychain")
			break
		}
		for _, s := range policy.Servers {
			if !isHubSpotServer(s) {
				continue
			}
			if err := installHubSpotCredsFromPolicy(policy); err != nil {
				fmt.Fprintln(os.Stderr, err)
				return 1
			}
			fmt.Fprintln(os.Stderr, "✓ installed HubSpot OAuth client into Keychain")
			break
		}
	}

	result, warnings, err := claude3p.Sync(policy, helperDir, keychainChecker())
	if err != nil {
		fmt.Fprintln(os.Stderr, err)
		return 1
	}

	for _, w := range warnings {
		fmt.Fprintln(os.Stderr, "WARN:", w)
	}

	if dry {
		fmt.Printf("# %s\n", result.ConfigLibraryPath)
		fmt.Println(string(result.ConfigLibraryJSON))
		return 0
	}

	if err := result.Apply(); err != nil {
		fmt.Fprintln(os.Stderr, "write failed:", err)
		return 1
	}
	fmt.Fprintf(os.Stderr, "✓ wrote %s\n", result.ConfigLibraryPath)
	printSyncLoginReminder(policy)
	return 0
}

func isSlackServer(s claude3p.ServerPolicy) bool {
	return strings.Contains(strings.ToLower(s.URL), "mcp.slack.com")
}

func isHubSpotServer(s claude3p.ServerPolicy) bool {
	return hubspotmcp.IsRemote(s.URL)
}

func slackClientFromPolicy(s claude3p.ServerPolicy) (clientID, clientSecret string) {
	if m, ok := s.OAuth.(map[string]interface{}); ok {
		if v, ok := m["client_id"].(string); ok {
			clientID = v
		} else if v, ok := m["clientId"].(string); ok {
			clientID = v
		}
		if v, ok := m["client_secret"].(string); ok {
			clientSecret = v
		} else if v, ok := m["clientSecret"].(string); ok {
			clientSecret = v
		}
	}
	if clientID == "" {
		clientID = strings.TrimSpace(s.ClientID)
	}
	if clientSecret == "" {
		clientSecret = strings.TrimSpace(s.ClientSecret)
	}
	return strings.TrimSpace(clientID), strings.TrimSpace(clientSecret)
}

func hubspotClientFromPolicy(s claude3p.ServerPolicy) (clientID, clientSecret string) {
	return slackClientFromPolicy(s)
}

func policyHasSlack(policy *claude3p.PolicyFile) bool {
	if policy == nil {
		return false
	}
	for _, s := range policy.Servers {
		if isSlackServer(s) {
			return true
		}
	}
	return false
}

func policyHasHubSpot(policy *claude3p.PolicyFile) bool {
	if policy == nil {
		return false
	}
	for _, s := range policy.Servers {
		if isHubSpotServer(s) {
			return true
		}
	}
	return false
}

func policyHasGoogle(policy *claude3p.PolicyFile) bool {
	if policy == nil {
		return false
	}
	for _, s := range policy.Servers {
		if strings.TrimSpace(s.GoogleService) != "" {
			return true
		}
	}
	return false
}

func policyGoogleScopes(policy *claude3p.PolicyFile) string {
	if policy == nil {
		return ""
	}
	var parts []string
	for _, s := range policy.Servers {
		gs := strings.TrimSpace(s.GoogleService)
		if gs == "" {
			continue
		}
		gs = strings.TrimPrefix(gs, "google-")
		if s.Scope != "" {
			parts = append(parts, s.Scope)
			continue
		}
		svc, err := googleworkspace.Lookup(gs)
		if err == nil {
			parts = append(parts, svc.DefaultScopes)
		}
	}
	return googleworkspace.MergeScopeStrings(parts...)
}

// keychainChecker returns a function that reports whether the Keychain has an
// OAuth client secret for a given Google service.
func keychainChecker() claude3p.KeychainChecker {
	return func(serviceID string) bool {
		svc, err := googleworkspace.Lookup(serviceID)
		if err != nil {
			return false
		}
		creds, err := googleworkspace.LoadClientCredentials(svc)
		return err == nil && creds.ClientSecret != ""
	}
}

func usage() {
	fmt.Fprint(os.Stderr, `ally3p — manage Claude Cowork 3P from a YAML policy file

`)
	printClaudeUsage()
	fmt.Fprint(os.Stderr, `
Other:
  prereq [--dir ./bin]             Install helper binaries (google, slack, hubspot)
  help                             Show this message

Policy file format: see README or run sync --dry to preview.
`)
}

func printSyncLoginReminder(policy *claude3p.PolicyFile) {
	var need []string
	if policyHasGoogle(policy) {
		need = append(need, "google")
	}
	if policyHasSlack(policy) {
		need = append(need, "slack")
	}
	if policyHasHubSpot(policy) {
		need = append(need, "hubspot")
	}
	if len(need) == 0 {
		return
	}
	fmt.Fprintf(os.Stderr, "OAuth tokens: run `ally3p claude login %s` when ready (sync does not open a browser).\n", strings.Join(need, ", "))
}

func printClaudeUsage() {
	fmt.Fprint(os.Stderr, `Claude commands:
  claude sync <policy.yaml>              Apply policy → configLibrary + Keychain (no browser OAuth)
  claude sync <policy.yaml> --dry        Preview JSON (no writes)

Login (browser OAuth / tokens):
  claude login <policy.yaml>             Sign in to all OAuth services in policy
  claude login slack [policy.yaml]       Slack (headersHelper)
  claude login hubspot [policy.yaml]     HubSpot (headersHelper + PKCE)
  claude login google [policy.yaml]      Google (unified account, all policy scopes)
  claude login figma [policy.yaml]       Info: auth via Claude Connect after sync
  claude login posthog [policy.yaml]     Info: auth via Claude Connect after sync

Examples:
  ally3p prereq --dir ./bin
  ally3p claude sync ally.yaml --dry
  ally3p claude sync ally.yaml
  ally3p claude login slack ally.yaml
  ally3p claude login hubspot ally.yaml
  ally3p claude login google ally.yaml
  ally3p claude login ally.yaml
`)
}

func printLoginUsage() {
	fmt.Fprint(os.Stderr, `usage: ally3p claude login <service> [policy.yaml]
       ally3p claude login [policy.yaml]   # all OAuth services in policy

Services:
  slack      Browser OAuth → Keychain (needs oauth client_id/secret in policy)
  hubspot    Browser OAuth + PKCE → Keychain (needs MCP Auth App creds in policy)
  google     Browser OAuth → Keychain (all google_service scopes in policy)
  figma      No login here — sync, restart Claude, click Connect
  posthog    No login here — sync, restart Claude, click Connect
  all        slack + hubspot + google from policy (default when policy.yaml given)

Examples:
  ally3p claude login slack ally.yaml
  ally3p claude login hubspot ally.yaml
  ally3p claude login google ally.yaml
  ally3p claude login ally.yaml
`)
}
