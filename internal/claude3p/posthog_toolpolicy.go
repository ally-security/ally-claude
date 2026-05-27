package claude3p

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/url"
	"strings"
	"time"
)

const posthogToolDefinitionsURL = "https://raw.githubusercontent.com/PostHog/posthog/master/services/mcp/schema/tool-definitions-all.json"

type posthogToolDefinition struct {
	Feature     string `json:"feature"`
	Annotations struct {
		ReadOnlyHint bool `json:"readOnlyHint"`
	} `json:"annotations"`
}

func isPostHogMCP(url string) bool {
	return strings.Contains(strings.ToLower(strings.TrimSpace(url)), "mcp.posthog.com")
}

func posthogParseFeaturesParam(mcpURL string) (map[string]struct{}, error) {
	u, err := url.Parse(strings.TrimSpace(mcpURL))
	if err != nil {
		return nil, fmt.Errorf("parse posthog mcp url: %w", err)
	}
	raw := strings.TrimSpace(u.Query().Get("features"))
	if raw == "" {
		return nil, nil
	}
	set := make(map[string]struct{})
	for _, part := range strings.Split(raw, ",") {
		part = posthogNormalizeFeatureName(part)
		if part != "" {
			set[part] = struct{}{}
		}
	}
	return set, nil
}

func posthogNormalizeFeatureName(s string) string {
	s = strings.TrimSpace(strings.ToLower(s))
	return strings.ReplaceAll(s, "-", "_")
}

func posthogFeatureMatches(feature string, allowed map[string]struct{}) bool {
	if len(allowed) == 0 {
		return true
	}
	_, ok := allowed[posthogNormalizeFeatureName(feature)]
	return ok
}

func posthogFetchToolDefinitions() (map[string]posthogToolDefinition, error) {
	client := &http.Client{Timeout: 30 * time.Second}
	resp, err := client.Get(posthogToolDefinitionsURL)
	if err != nil {
		return nil, fmt.Errorf("fetch posthog tool definitions: %w", err)
	}
	defer resp.Body.Close()
	if resp.StatusCode != http.StatusOK {
		return nil, fmt.Errorf("fetch posthog tool definitions: HTTP %d", resp.StatusCode)
	}
	var defs map[string]posthogToolDefinition
	if err := json.NewDecoder(resp.Body).Decode(&defs); err != nil {
		return nil, fmt.Errorf("decode posthog tool definitions: %w", err)
	}
	return defs, nil
}
