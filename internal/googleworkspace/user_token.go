package googleworkspace

import (
	"encoding/json"
	"fmt"
	"os"
	"strings"
)

// SharedUserKeychainAccount stores one OAuth user token for all Google Workspace MCP services.
const SharedUserKeychainAccount = "google-user"

// UnifiedCallbackPort is the redirect port used for the combined Google sign-in flow.
const UnifiedCallbackPort = 53281

// SharedUserStore is the keychain location for the shared user token.
func SharedUserStore() Store {
	return Store{
		Service: KeychainServiceNameForStore(),
		Account: SharedUserKeychainAccount,
	}
}

// MergeScopeStrings unions space-separated OAuth scopes (deduped).
func MergeScopeStrings(parts ...string) string {
	seen := map[string]bool{}
	var out []string
	for _, part := range parts {
		for _, s := range strings.Fields(part) {
			if s == "" || seen[s] {
				continue
			}
			seen[s] = true
			out = append(out, s)
		}
	}
	return strings.Join(out, " ")
}

// MergedScopesFromClaude3P unions scopes for all Google MCP entries in the active Claude config.
func MergedScopesFromClaude3P() (string, error) {
	dir, err := claude3pConfigLibraryDir()
	if err != nil {
		return "", err
	}
	cfgPath, err := activeClaude3PConfigPath(dir)
	if err != nil {
		return "", err
	}
	data, err := os.ReadFile(cfgPath)
	if err != nil {
		return "", err
	}
	scopes, err := mergedScopesFromConfigJSON(data)
	if err != nil {
		return "", err
	}
	if scopes != "" {
		return scopes, nil
	}
	return MergeScopeStrings(scopeListFromCatalog()...), nil
}

func mergedScopesFromConfigJSON(data []byte) (string, error) {
	var doc struct {
		Managed []struct {
			Name          string `json:"name"`
			URL           string `json:"url"`
			Scope         string `json:"scope"`
			HeadersHelper string `json:"headersHelper"`
		} `json:"managedMcpServers"`
	}
	if err := json.Unmarshal(data, &doc); err != nil {
		return "", err
	}
	var parts []string
	for _, m := range doc.Managed {
		if m.HeadersHelper == "" && !looksLikeGoogleMCP(m.Name, m.URL) {
			continue
		}
		if !looksLikeGoogleMCP(m.Name, m.URL) {
			continue
		}
		scope := m.Scope
		if scope == "" {
			if svc, ok := matchServiceByNameOrURL(m.Name, m.URL); ok {
				scope = svc.DefaultScopes
			}
		}
		if scope != "" {
			parts = append(parts, scope)
		}
	}
	return MergeScopeStrings(parts...), nil
}

func looksLikeGoogleMCP(name, url string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if strings.HasPrefix(n, "google-") || n == "gmail" || n == "drive" || n == "calendar" || n == "chat" || n == "people" {
		return true
	}
	u := strings.ToLower(url)
	return strings.Contains(u, "googleapis.com/mcp") || strings.Contains(u, "gmailmcp.googleapis.com") ||
		strings.Contains(u, "drivemcp.googleapis.com") || strings.Contains(u, "calendarmcp.googleapis.com") ||
		strings.Contains(u, "chatmcp.googleapis.com")
}

func matchServiceByNameOrURL(name, url string) (Service, bool) {
	for _, s := range Catalog {
		if connectorMatchesService(name, url, s) {
			return s, true
		}
	}
	return Service{}, false
}

func scopeListFromCatalog() []string {
	parts := make([]string, len(Catalog))
	for i, s := range Catalog {
		parts[i] = s.DefaultScopes
	}
	return parts
}

// EnsureUnifiedLogin runs one browser sign-in for all Google services if no shared token exists.
func EnsureUnifiedLogin(extraScopes ...string) error {
	store := SharedUserStore()
	if tok, err := store.Load(); err == nil && tok.RefreshToken != "" {
		return nil
	}
	return unifiedLogin(MergeScopeStrings(extraScopes...), false)
}

// ForceUnifiedLogin always opens the browser OAuth flow (even if a token already exists).
func ForceUnifiedLogin(extraScopes ...string) error {
	return unifiedLogin(MergeScopeStrings(extraScopes...), true)
}

func unifiedLogin(extraScopes string, force bool) error {
	store := SharedUserStore()
	if !force {
		if tok, err := store.Load(); err == nil && tok.RefreshToken != "" {
			return nil
		}
	}

	svc, err := Lookup("gmail")
	if err != nil {
		return err
	}
	cfg, err := LoadConfig(svc)
	if err != nil {
		return err
	}

	merged, err := MergedScopesFromClaude3P()
	if err != nil {
		merged = svc.DefaultScopes
	}
	if strings.TrimSpace(extraScopes) != "" {
		cfg.Scope = extraScopes
	} else {
		cfg.Scope = merged
	}
	cfg.CallbackPort = UnifiedCallbackPort

	fmt.Fprintf(stderr, "Sign in with Google once for Gmail, Drive, Calendar, and other Workspace MCP services…\n")
	if err := Login(cfg, store); err != nil {
		return err
	}
	fmt.Fprintf(stderr, "✓ saved shared Google token to keychain (%s / %s)\n", store.Service, store.Account)
	return nil
}
