// Package claude3p reads a YAML policy file and writes the Claude 3P configLibrary JSON.
package claude3p

import (
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/anthropics/google-workspace-mcp-auth/internal/figmamcp"
	"github.com/anthropics/google-workspace-mcp-auth/internal/hubspotmcp"
)

// PolicyFile is the user-facing YAML schema.
// Example:
//
//	inference:
//	  provider: bedrock
//	  bedrock_region: us-east-1
//	  bedrock_token: "..."
//
//	banner:
//	  enabled: false
//
//	servers:
//	  - name: linear
//	    url: https://mcp.linear.app/mcp
//	    transport: http
//	    oauth: true
//
//	  - name: posthog
//	    url: https://mcp.posthog.com/mcp
//	    transport: http
//	    oauth: true
//
//	  - name: google-gmail
//	    google_service: gmail          # auto-wires headersHelper if Keychain has client secret
//	    client_id: "...apps.googleusercontent.com"
//	    callback_port: 53281
//	    scope: "https://www.googleapis.com/auth/gmail.readonly ..."
type PolicyFile struct {
	Inference *InferencePolicy `yaml:"inference"`
	Banner    *BannerPolicy    `yaml:"banner"`
	Servers []ServerPolicy `yaml:"servers"`
}

type InferencePolicy struct {
	Provider    string `yaml:"provider"`
	CredKind    string `yaml:"cred_kind"`
	Region      string `yaml:"bedrock_region"`
	BearerToken string `yaml:"bedrock_token"`
}

type BannerPolicy struct {
	Enabled         bool   `yaml:"enabled"`
	BackgroundColor string `yaml:"background_color"`
	TextColor       string `yaml:"text_color"`
}

type ServerPolicy struct {
	Name   string `yaml:"name"`
	URL    string `yaml:"url"`
	Source string `yaml:"source"`

	// Generic OAuth (e.g. Linear: true; Slack: client_id + callback_port).
	OAuth interface{} `yaml:"oauth"` // bool or object

	// Slack/Cursor-style auth block (maps to oauth.clientId in Claude config).
	Auth map[string]interface{} `yaml:"auth"`

	Transport string            `yaml:"transport"` // default http when url is set
	Headers   map[string]string `yaml:"headers"`

	// Google-specific: set google_service to one of gmail/drive/calendar/chat/people.
	// The binary path, URL, and default ports/scopes are auto-filled.
	GoogleService string `yaml:"google_service"`

	// Optional overrides when google_service is set.
	ClientID     string `yaml:"client_id"`
	ClientSecret string `yaml:"client_secret"` // prefer Keychain; falls back to this
	CallbackPort int    `yaml:"callback_port"`
	Scope        string `yaml:"scope"`

	// Path to the headersHelper binary (auto-detected when google_service is set).
	HeadersHelper    string `yaml:"headers_helper"`
	HeadersHelperTTL int    `yaml:"headers_helper_ttl_sec"`

	// Per-tool approval locks (tool name → allow | ask | blocked).
	ToolPolicy map[string]string `yaml:"tool_policy"`
}

// ParsePolicyFile reads a YAML policy file.
func ParsePolicyFile(path string) (*PolicyFile, error) {
	data, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("read policy file %s: %w", path, err)
	}
	var p PolicyFile
	if err := yaml.Unmarshal(data, &p); err != nil {
		return nil, fmt.Errorf("parse policy file %s: %w", path, err)
	}
	return &p, nil
}

// SyncResult describes the files that sync will write (or would write in dry mode).
type SyncResult struct {
	ConfigLibraryPath string
	ConfigLibraryJSON []byte
}

// Sync computes the configLibrary JSON from the policy.
// helperBinaryDir is the directory containing the google-workspace-mcp-auth[-service] binaries.
// Pass "" to auto-detect (checks PATH locations).
func Sync(policy *PolicyFile, helperBinaryDir string, keychainChecker KeychainChecker) (*SyncResult, []string, error) {
	var warnings []string

	config := make(map[string]interface{})

	// --- Inference ---
	if inf := policy.Inference; inf != nil {
		provider := inf.Provider
		if provider == "" {
			provider = "bedrock"
		}
		config["inferenceProvider"] = provider
		if inf.CredKind != "" {
			config["inferenceCredentialKind"] = inf.CredKind
		}
		if inf.Region != "" {
			config["inferenceBedrockRegion"] = inf.Region
		}
		if inf.BearerToken != "" {
			config["inferenceBedrockBearerToken"] = inf.BearerToken
		}
	}

	// --- Banner ---
	if b := policy.Banner; b != nil {
		banner := map[string]interface{}{
			"enabled": b.Enabled,
		}
		if b.BackgroundColor != "" {
			banner["backgroundColor"] = b.BackgroundColor
		}
		if b.TextColor != "" {
			banner["textColor"] = b.TextColor
		}
		config["banner"] = banner
	}

	// --- Servers ---
	var servers []interface{}
	for _, srv := range policy.Servers {
		entry, warn, err := buildServerEntry(srv, helperBinaryDir, keychainChecker)
		if err != nil {
			return nil, warnings, fmt.Errorf("server %q: %w", srv.Name, err)
		}
		warnings = append(warnings, warn...)
		servers = append(servers, entry)
	}
	if servers != nil {
		config["managedMcpServers"] = servers
	}

	data, err := json.MarshalIndent(config, "", "  ")
	if err != nil {
		return nil, warnings, err
	}
	data = append(data, '\n')

	cfgPath, err := activeConfigLibraryPath()
	if err != nil {
		return nil, warnings, fmt.Errorf("locate Claude 3P configLibrary: %w", err)
	}

	return &SyncResult{
		ConfigLibraryPath: cfgPath,
		ConfigLibraryJSON: data,
	}, warnings, nil
}

// Apply writes the result to disk.
func (r *SyncResult) Apply() error {
	if err := os.MkdirAll(filepath.Dir(r.ConfigLibraryPath), 0o700); err != nil {
		return err
	}
	return os.WriteFile(r.ConfigLibraryPath, r.ConfigLibraryJSON, 0o600)
}

// KeychainChecker reports whether an OAuth client secret exists in Keychain for a Google service id.
type KeychainChecker func(serviceID string) bool

func buildServerEntry(srv ServerPolicy, helperDir string, kc KeychainChecker) (map[string]interface{}, []string, error) {
	var warnings []string
	entry := map[string]interface{}{}

	source := srv.Source
	if source == "" {
		source = "user"
	}

	if srv.GoogleService != "" {
		// --- Google Workspace service ---
		gsvc, err := lookupGoogleService(srv.GoogleService)
		if err != nil {
			return nil, warnings, err
		}

		if strings.TrimSpace(srv.ClientID) == "" {
			return nil, warnings, fmt.Errorf("google_service %q requires client_id", gsvc.ID)
		}

		name := srv.Name
		if name == "" {
			name = "google-" + gsvc.ID
		}
		url := srv.URL
		if url == "" {
			url = gsvc.MCPURL
		}
		port := srv.CallbackPort
		if port == 0 {
			port = gsvc.DefaultCallbackPort
		}
		scope := srv.Scope
		if scope == "" {
			scope = gsvc.DefaultScopes
		}

		entry["name"] = name
		entry["source"] = source
		entry["transport"] = "http"
		entry["url"] = url

		// Determine if we can wire headersHelper.
		helperPath := srv.HeadersHelper
		if helperPath == "" {
			helperPath = resolveHelperBinary("google-workspace-mcp-auth-"+gsvc.ID, helperDir)
		}

		hasKeychainSecret := kc != nil && kc(gsvc.ID)
		hasSecretProvidedInYAML := strings.TrimSpace(srv.ClientSecret) != ""

		if helperPath == "" {
			warnings = append(warnings, fmt.Sprintf(
				"[%s] google-workspace-mcp-auth-%s not found (checked helper-dir, alongside ally3p, /usr/local/bin, /opt/homebrew/bin, PATH) — using oauth:true (may not work for Google); install and re-sync",
				name, gsvc.ID,
			))
			entry["oauth"] = true
		} else {
			entry["headersHelper"] = helperPath
			ttl := srv.HeadersHelperTTL
			if ttl == 0 {
				ttl = 300
			}
			entry["headersHelperTtlSec"] = ttl

			// Claude rejects entries that have both headersHelper and oauth.
			// Store clientId/callbackPort/scope as top-level fields instead —
			// Claude ignores unknown fields and our binary reads them from there.
			entry["clientId"] = strings.TrimSpace(srv.ClientID)
			entry["callbackPort"] = port
			entry["scope"] = scope

			if !hasKeychainSecret && !hasSecretProvidedInYAML {
				warnings = append(warnings, fmt.Sprintf(
					"[%s] no client secret found in Keychain — provide client_secret in YAML (or ensure it is already installed in Keychain)",
					name,
				))
			}
		}
		warn, err := attachToolPolicy(entry, srv)
		if err != nil {
			return nil, warnings, err
		}
		warnings = append(warnings, warn...)
		return entry, warnings, nil
	}

	// --- Generic server ---
	if strings.TrimSpace(srv.Name) == "" {
		return nil, warnings, fmt.Errorf("server entry requires name")
	}
	if strings.TrimSpace(srv.URL) == "" {
		return nil, warnings, fmt.Errorf("server %q requires url", srv.Name)
	}

	entry["name"] = srv.Name
	entry["source"] = source
	entry["url"] = srv.URL
	transport := strings.TrimSpace(srv.Transport)
	if transport == "" {
		transport = "http"
	}
	entry["transport"] = transport
	if len(srv.Headers) > 0 {
		entry["headers"] = srv.Headers
	}

	if isSlackMCP(srv.URL) {
		if entrySlack, warn, err := buildSlackEntry(srv, helperDir); err != nil {
			return nil, warnings, err
		} else {
			warnings = append(warnings, warn...)
			for k, v := range entrySlack {
				entry[k] = v
			}
			warn, err := attachToolPolicy(entry, srv)
			if err != nil {
				return nil, warnings, err
			}
			warnings = append(warnings, warn...)
			return entry, warnings, nil
		}
	}

	if hubspotmcp.IsRemote(srv.URL) {
		if entryHubSpot, warn, err := buildHubSpotEntry(srv, helperDir); err != nil {
			return nil, warnings, err
		} else {
			warnings = append(warnings, warn...)
			for k, v := range entryHubSpot {
				entry[k] = v
			}
			warn, err := attachToolPolicy(entry, srv)
			if err != nil {
				return nil, warnings, err
			}
			warnings = append(warnings, warn...)
			return entry, warnings, nil
		}
	}

	if figmamcp.IsDesktop(srv.URL) {
		warnings = append(warnings, fmt.Sprintf(
			"[%s] Figma desktop MCP requires the Figma desktop app running with MCP enabled (no OAuth)",
			srv.Name,
		))
	}
	if len(srv.Headers) > 0 && buildOAuthEntry(srv) != nil {
		warnings = append(warnings, fmt.Sprintf(
			"[%s] Claude 3P does not allow headers and oauth on the same MCP server — use one or the other",
			srv.Name,
		))
	}
	if oauth := buildOAuthEntry(srv); oauth != nil {
		entry["oauth"] = oauth
	}
	if srv.HeadersHelper != "" {
		entry["headersHelper"] = srv.HeadersHelper
		ttl := srv.HeadersHelperTTL
		if ttl == 0 {
			ttl = 300
		}
		entry["headersHelperTtlSec"] = ttl
	}
	warn, err := attachToolPolicy(entry, srv)
	if err != nil {
		return nil, warnings, err
	}
	warnings = append(warnings, warn...)
	return entry, warnings, nil
}

func isSlackMCP(url string) bool {
	return strings.Contains(strings.ToLower(url), "mcp.slack.com")
}

func buildSlackEntry(srv ServerPolicy, helperDir string) (map[string]interface{}, []string, error) {
	var warnings []string
	entry := map[string]interface{}{}

	helperPath := srv.HeadersHelper
	if helperPath == "" {
		helperPath = resolveHelperBinary("slack-mcp-auth", helperDir)
	}
	if helperPath == "" {
		warnings = append(warnings, fmt.Sprintf(
			"[%s] slack-mcp-auth not found — falling back to oauth (may fail in Claude); run: ally3p prereq",
			srv.Name,
		))
		if oauth := buildOAuthEntry(srv); oauth != nil {
			entry["oauth"] = oauth
		}
		return entry, warnings, nil
	}

	entry["headersHelper"] = helperPath
	ttl := srv.HeadersHelperTTL
	if ttl == 0 {
		ttl = 300
	}
	entry["headersHelperTtlSec"] = ttl
	return entry, warnings, nil
}

func buildHubSpotEntry(srv ServerPolicy, helperDir string) (map[string]interface{}, []string, error) {
	var warnings []string
	entry := map[string]interface{}{}

	helperPath := srv.HeadersHelper
	if helperPath == "" {
		helperPath = resolveHelperBinary("hubspot-mcp-auth", helperDir)
	}
	if helperPath == "" {
		warnings = append(warnings, fmt.Sprintf(
			"[%s] hubspot-mcp-auth not found — falling back to oauth (may fail in Claude); run: ally3p prereq",
			srv.Name,
		))
		if oauth := buildOAuthEntry(srv); oauth != nil {
			entry["oauth"] = oauth
		}
		return entry, warnings, nil
	}

	entry["headersHelper"] = helperPath
	ttl := srv.HeadersHelperTTL
	if ttl == 0 {
		ttl = 300
	}
	entry["headersHelperTtlSec"] = ttl
	return entry, warnings, nil
}

func buildOAuthEntry(srv ServerPolicy) interface{} {
	if srv.OAuth != nil {
		if b, ok := srv.OAuth.(bool); ok {
			return b
		}
		if m, ok := srv.OAuth.(map[string]interface{}); ok {
			return normalizeOAuthMap(m, srv.URL, srv.CallbackPort, srv.ClientSecret)
		}
		if m, ok := srv.OAuth.(map[interface{}]interface{}); ok {
			flat := make(map[string]interface{}, len(m))
			for k, v := range m {
				if s, ok := k.(string); ok {
					flat[s] = v
				}
			}
			return normalizeOAuthMap(flat, srv.URL, srv.CallbackPort, srv.ClientSecret)
		}
	}

	if cid := authClientID(srv.Auth); cid != "" {
		return slackOAuthObject(cid, authClientSecret(srv.Auth, srv.ClientSecret), srv.URL, srv.CallbackPort)
	}

	if strings.TrimSpace(srv.ClientID) != "" && srv.GoogleService == "" {
		return slackOAuthObject(strings.TrimSpace(srv.ClientID), strings.TrimSpace(srv.ClientSecret), srv.URL, srv.CallbackPort)
	}

	return nil
}

func authClientID(auth map[string]interface{}) string {
	if auth == nil {
		return ""
	}
	for _, key := range []string{"CLIENT_ID", "client_id", "clientId"} {
		if v, ok := auth[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func authClientSecret(auth map[string]interface{}, fallback string) string {
	if s := strings.TrimSpace(fallback); s != "" {
		return s
	}
	if auth == nil {
		return ""
	}
	for _, key := range []string{"CLIENT_SECRET", "client_secret", "clientSecret"} {
		if v, ok := auth[key].(string); ok && strings.TrimSpace(v) != "" {
			return strings.TrimSpace(v)
		}
	}
	return ""
}

func slackOAuthObject(clientID, clientSecret, url string, callbackPort int) map[string]interface{} {
	oauth := map[string]interface{}{"clientId": clientID}
	if clientSecret != "" {
		oauth["clientSecret"] = clientSecret
	}
	port := callbackPort
	if port == 0 && strings.Contains(url, "mcp.slack.com") {
		port = 3118
	}
	if port != 0 {
		oauth["callbackPort"] = port
	}
	return oauth
}

func normalizeOAuthMap(m map[string]interface{}, url string, fallbackPort int, fallbackSecret string) map[string]interface{} {
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		switch k {
		case "client_id", "clientId", "CLIENT_ID":
			if s, ok := v.(string); ok {
				out["clientId"] = strings.TrimSpace(s)
			}
		case "client_secret", "clientSecret", "CLIENT_SECRET":
			if s, ok := v.(string); ok {
				out["clientSecret"] = strings.TrimSpace(s)
			}
		case "callback_port", "callbackPort":
			out["callbackPort"] = v
		case "tenant_id", "tenantId":
			out["tenantId"] = v
		case "scope":
			out["scope"] = v
		case "callback_host", "callbackHost":
			out["callbackHost"] = v
		default:
			out[k] = v
		}
	}
	if _, ok := out["clientId"]; !ok {
		if cid := authClientID(m); cid != "" {
			out["clientId"] = cid
		}
	}
	if _, ok := out["clientSecret"]; !ok {
		if secret := authClientSecret(m, fallbackSecret); secret != "" {
			out["clientSecret"] = secret
		}
	}
	if _, ok := out["callbackPort"]; !ok {
		if fallbackPort != 0 {
			out["callbackPort"] = fallbackPort
		} else if strings.Contains(url, "mcp.slack.com") {
			out["callbackPort"] = 3118
		} else if strings.Contains(url, "mcp.hubspot.com") {
			out["callbackPort"] = hubspotmcp.DefaultCallbackPort
		}
	}
	return out
}

func resolveHelperBinary(name, dir string) string {
	candidates := []string{}
	if dir != "" {
		candidates = append(candidates, filepath.Join(dir, name))
	}
	if exe, err := os.Executable(); err == nil {
		candidates = append(candidates, filepath.Join(filepath.Dir(exe), name))
	}
	// Common install locations
	candidates = append(candidates,
		"/usr/local/bin/"+name,
		"/opt/homebrew/bin/"+name,
	)
	for _, p := range candidates {
		if _, err := os.Stat(p); err == nil {
			if abs, err := filepath.Abs(p); err == nil {
				return abs
			}
			return p
		}
	}
	if p, err := exec.LookPath(name); err == nil {
		return p
	}
	return ""
}

func activeConfigLibraryPath() (string, error) {
	base, err := claude3pAppSupportDir()
	if err != nil {
		return "", err
	}
	libDir := filepath.Join(base, "configLibrary")
	metaBytes, err := os.ReadFile(filepath.Join(libDir, "_meta.json"))
	if err != nil {
		return "", fmt.Errorf("read _meta.json: %w", err)
	}
	var meta struct {
		ActiveConfigID string `json:"activeConfigId"`
		AppliedID      string `json:"appliedId"`
	}
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return "", fmt.Errorf("parse _meta.json: %w", err)
	}
	id := meta.ActiveConfigID
	if id == "" {
		id = meta.AppliedID
	}
	if id == "" {
		return "", fmt.Errorf("_meta.json has no activeConfigId or appliedId")
	}
	return filepath.Join(libDir, id+".json"), nil
}

func claude3pAppSupportDir() (string, error) {
	if d := os.Getenv("CLAUDE_3P_HOME"); d != "" {
		return d, nil
	}
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	switch runtime.GOOS {
	case "darwin":
		return filepath.Join(home, "Library", "Application Support", "Claude-3p"), nil
	case "windows":
		return filepath.Join(os.Getenv("LOCALAPPDATA"), "Claude-3p"), nil
	default:
		return filepath.Join(home, ".config", "Claude-3p"), nil
	}
}

// googleService is a minimal mirror of googleworkspace.Service to avoid import cycles.
type googleService struct {
	ID                  string
	MCPURL              string
	DefaultScopes       string
	DefaultCallbackPort int
}

var googleCatalog = []googleService{
	{"gmail", "https://gmailmcp.googleapis.com/mcp/v1",
		"https://www.googleapis.com/auth/gmail.readonly https://www.googleapis.com/auth/gmail.compose", 53281},
	{"drive", "https://drivemcp.googleapis.com/mcp/v1",
		"https://www.googleapis.com/auth/drive.readonly https://www.googleapis.com/auth/drive.file", 53282},
	{"calendar", "https://calendarmcp.googleapis.com/mcp/v1",
		"https://www.googleapis.com/auth/calendar.calendarlist.readonly https://www.googleapis.com/auth/calendar.events.freebusy https://www.googleapis.com/auth/calendar.events.readonly", 53283},
	{"chat", "https://chatmcp.googleapis.com/mcp/v1",
		"https://www.googleapis.com/auth/chat.spaces.readonly https://www.googleapis.com/auth/chat.memberships.readonly https://www.googleapis.com/auth/chat.messages.readonly https://www.googleapis.com/auth/chat.messages.create https://www.googleapis.com/auth/chat.users.readstate.readonly", 53284},
	{"people", "https://people.googleapis.com/mcp/v1",
		"https://www.googleapis.com/auth/directory.readonly https://www.googleapis.com/auth/userinfo.profile https://www.googleapis.com/auth/contacts.readonly", 53285},
}

func lookupGoogleService(id string) (googleService, error) {
	id = strings.ToLower(strings.TrimSpace(id))
	id = strings.TrimPrefix(id, "google-")
	for _, s := range googleCatalog {
		if s.ID == id {
			return s, nil
		}
	}
	ids := make([]string, len(googleCatalog))
	for i, s := range googleCatalog {
		ids[i] = s.ID
	}
	return googleService{}, fmt.Errorf("unknown google_service %q (valid: %s)", id, strings.Join(ids, ", "))
}
