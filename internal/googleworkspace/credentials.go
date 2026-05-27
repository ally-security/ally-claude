package googleworkspace

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
)

// ClientCredentials is the OAuth app identity (deployed by IT, not end users).
type ClientCredentials struct {
	ClientID     string
	ClientSecret string
	CallbackPort int // 0 = use service default
	Scope        string
	Source       string // for error messages
}

type credentialsFile struct {
	ClientID     string                     `json:"clientId"`
	ClientSecret string                     `json:"clientSecret"`
	CallbackPort int                        `json:"callbackPort,omitempty"`
	Scope        string                     `json:"scope,omitempty"`
	Services     map[string]credentialsFile `json:"services,omitempty"`
}

// LoadClientCredentials resolves OAuth app credentials without end-user env vars.
//
// Order:
//  1. macOS Keychain (install-credentials, once per machine)
//  2. Claude 3P configLibrary (clientId, callbackPort, scope; secret optional if in Keychain)
//  3. Enterprise drop file (google-workspace-oauth.json)
//  4. GOOGLE_CLIENT_ID / GOOGLE_CLIENT_SECRET (developer override only)
func LoadClientCredentials(svc Service) (ClientCredentials, error) {
	var c ClientCredentials

	if kc, err := credentialsFromKeychain(svc); err == nil {
		c = kc
	} else if !errorsIsNotFound(err) {
		return ClientCredentials{}, err
	}

	if hints, err := oauthHintsFromClaude3P(svc); err == nil {
		c = mergeClientCredentials(c, hints)
	}

	if c.ClientID != "" && c.ClientSecret != "" {
		return c, nil
	}

	if legacy, err := credentialsFromClaude3PFull(svc); err == nil {
		return legacy, nil
	}

	if c.ClientID != "" && c.ClientSecret == "" {
		return ClientCredentials{}, fmt.Errorf(
			"oauth.clientId is configured for %s but client secret is missing — run once on this Mac: google-workspace-mcp-auth install-credentials --client-id %q --client-secret SECRET",
			svc.ID, c.ClientID,
		)
	}

	if fileCreds, err := credentialsFromDropFile(svc); err == nil {
		return fileCreds, nil
	}

	if id, secret := os.Getenv("GOOGLE_CLIENT_ID"), os.Getenv("GOOGLE_CLIENT_SECRET"); id != "" && secret != "" {
		return ClientCredentials{
			ClientID:     id,
			ClientSecret: secret,
			Source:       "environment (dev only)",
		}, nil
	}

	return ClientCredentials{}, fmt.Errorf(
		"no OAuth client configured for %s — IT must run install-credentials on this Mac, deploy %s, or set managedMcpServers.oauth in Claude configLibrary",
		svc.ID,
		DefaultCredentialsPath(),
	)
}

func credentialsFromDropFile(svc Service) (ClientCredentials, error) {
	for _, path := range credentialsSearchPaths() {
		data, err := os.ReadFile(path)
		if err != nil {
			continue
		}
		var f credentialsFile
		if err := json.Unmarshal(data, &f); err != nil {
			return ClientCredentials{}, fmt.Errorf("%s: %w", path, err)
		}
		c := ClientCredentials{
			ClientID:     f.ClientID,
			ClientSecret: f.ClientSecret,
			CallbackPort: f.CallbackPort,
			Scope:        f.Scope,
			Source:       path,
		}
		if sub, ok := f.Services[svc.ID]; ok {
			if sub.ClientID != "" {
				c.ClientID = sub.ClientID
			}
			if sub.ClientSecret != "" {
				c.ClientSecret = sub.ClientSecret
			}
			if sub.CallbackPort != 0 {
				c.CallbackPort = sub.CallbackPort
			}
			if sub.Scope != "" {
				c.Scope = sub.Scope
			}
		}
		if c.ClientID == "" || c.ClientSecret == "" {
			continue
		}
		return c, nil
	}
	return ClientCredentials{}, os.ErrNotExist
}

func credentialsSearchPaths() []string {
	var paths []string
	if p := os.Getenv("GOOGLE_WORKSPACE_MCP_CREDENTIALS"); p != "" {
		paths = append(paths, p)
	}
	paths = append(paths, DefaultCredentialsPath())
	if exe, err := os.Executable(); err == nil {
		paths = append(paths, filepath.Join(filepath.Dir(exe), "google-workspace-oauth.json"))
	}
	if runtime.GOOS == "darwin" {
		paths = append(paths, "/Library/Application Support/Claude-3p/google-workspace-oauth.json")
	}
	return paths
}

// DefaultCredentialsPath is the enterprise OAuth client file IT can deploy.
func DefaultCredentialsPath() string {
	dir, err := claude3pAppSupportDir()
	if err != nil {
		return filepath.Join(".", "google-workspace-oauth.json")
	}
	return filepath.Join(dir, "google-workspace-oauth.json")
}

func oauthHintsFromClaude3P(svc Service) (ClientCredentials, error) {
	cfgPath, oauthMap, name, err := readClaude3POAuthMap(svc)
	if err != nil {
		return ClientCredentials{}, err
	}
	c := ClientCredentials{Source: cfgPath}
	if v, ok := oauthMap["clientId"].(string); ok {
		c.ClientID = v
	}
	if v, ok := oauthMap["clientSecret"].(string); ok {
		c.ClientSecret = v
	}
	if v, ok := oauthMap["scope"].(string); ok {
		c.Scope = v
	}
	if v, ok := oauthMap["callbackPort"].(float64); ok {
		c.CallbackPort = int(v)
	}
	if c.ClientID == "" && c.ClientSecret == "" && c.Scope == "" && c.CallbackPort == 0 {
		return ClientCredentials{}, fmt.Errorf("%q in %s has empty oauth object", name, cfgPath)
	}
	return c, nil
}

func credentialsFromClaude3PFull(svc Service) (ClientCredentials, error) {
	c, err := oauthHintsFromClaude3P(svc)
	if err != nil {
		return ClientCredentials{}, err
	}
	if c.ClientID == "" || c.ClientSecret == "" {
		return ClientCredentials{}, fmt.Errorf(
			"%q is missing oauth.clientId or oauth.clientSecret (use install-credentials for the secret)",
			svc.ID,
		)
	}
	return c, nil
}

func readClaude3POAuthMap(svc Service) (cfgPath string, oauthMap map[string]interface{}, name string, err error) {
	dir, err := claude3pConfigLibraryDir()
	if err != nil {
		return "", nil, "", err
	}
	cfgPath, err = activeClaude3PConfigPath(dir)
	if err != nil {
		return "", nil, "", err
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return "", nil, "", err
	}
	var doc struct {
		Managed []struct {
			Name  string      `json:"name"`
			URL   string      `json:"url"`
			OAuth interface{} `json:"oauth"`
			// Top-level fields written when headersHelper is used
			// (Claude rejects entries with both oauth and headersHelper).
			ClientID     string  `json:"clientId"`
			CallbackPort float64 `json:"callbackPort"`
			Scope        string  `json:"scope"`
		} `json:"managedMcpServers"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", nil, "", err
	}
	for _, m := range doc.Managed {
		if !connectorMatchesService(m.Name, m.URL, svc) {
			continue
		}
		// Prefer explicit oauth object; fall back to top-level fields
		// (written by ally3p sync when headersHelper is present).
		if oauthMap, ok := m.OAuth.(map[string]interface{}); ok {
			return cfgPath, oauthMap, m.Name, nil
		}
		if m.ClientID != "" {
			return cfgPath, map[string]interface{}{
				"clientId":     m.ClientID,
				"callbackPort": m.CallbackPort,
				"scope":        m.Scope,
			}, m.Name, nil
		}
		return cfgPath, nil, m.Name, fmt.Errorf(
			"%s entry %q has no clientId — re-run ally3p claude sync",
			cfgPath, m.Name,
		)
	}
	return "", nil, "", os.ErrNotExist
}

func connectorMatchesService(name, url string, svc Service) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	id := svc.ID
	if n == id || n == "google-"+id || n == "google_"+id {
		return true
	}
	if url != "" && strings.Contains(url, serviceURLFragment(id)) {
		return true
	}
	return false
}

func serviceURLFragment(id string) string {
	switch id {
	case "gmail":
		return "gmailmcp.googleapis.com"
	case "drive":
		return "drivemcp.googleapis.com"
	case "calendar":
		return "calendarmcp.googleapis.com"
	case "chat":
		return "chatmcp.googleapis.com"
	case "people":
		return "people.googleapis.com"
	default:
		return id
	}
}

func claude3pConfigLibraryDir() (string, error) {
	base, err := claude3pAppSupportDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(base, "configLibrary"), nil
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

func activeClaude3PConfigPath(configDir string) (string, error) {
	metaPath := filepath.Join(configDir, "_meta.json")
	metaBytes, err := os.ReadFile(metaPath)
	if err != nil {
		return "", err
	}
	var meta struct {
		ActiveConfigID string `json:"activeConfigId"`
		AppliedID      string `json:"appliedId"`
	}
	if err := json.Unmarshal(metaBytes, &meta); err != nil {
		return "", err
	}
	id := meta.ActiveConfigID
	if id == "" {
		id = meta.AppliedID
	}
	if id == "" {
		return "", fmt.Errorf("_meta.json has no activeConfigId or appliedId")
	}
	cfgPath := filepath.Join(configDir, id+".json")
	if _, err := os.Stat(cfgPath); err != nil {
		return "", err
	}
	return cfgPath, nil
}

// WriteCredentialsDropFile is used by `install-credentials --file` for non-Keychain deployment.
func WriteCredentialsDropFile(path string, clientID, clientSecret string) error {
	if path == "" {
		path = DefaultCredentialsPath()
	}
	payload := credentialsFile{
		ClientID:     clientID,
		ClientSecret: clientSecret,
	}
	data, err := json.MarshalIndent(payload, "", "  ")
	if err != nil {
		return err
	}
	if err := os.MkdirAll(filepath.Dir(path), 0o700); err != nil {
		return err
	}
	return os.WriteFile(path, append(data, '\n'), 0o600)
}

func mergeCredentialsIntoConfig(svc Service, c ClientCredentials) Config {
	port := svc.DefaultCallbackPort
	if c.CallbackPort != 0 {
		port = c.CallbackPort
	}
	scope := svc.DefaultScopes
	if c.Scope != "" {
		scope = c.Scope
	}
	if p := os.Getenv("CALLBACK_PORT"); p != "" {
		if n, err := strconv.Atoi(p); err == nil {
			port = n
		}
	}
	if scopeEnv := os.Getenv("GOOGLE_MCP_SCOPE"); scopeEnv != "" {
		scope = scopeEnv
	}
	return Config{
		Service:      svc,
		ClientID:     c.ClientID,
		ClientSecret: c.ClientSecret,
		CallbackPort: port,
		Scope:        scope,
	}
}
