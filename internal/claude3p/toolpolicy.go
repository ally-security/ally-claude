package claude3p

import (
	"fmt"
	"strings"
)

const toolPolicyDefaultUnlisted = "ask"

// mergeToolPolicy builds configLibrary toolPolicy from ally.yaml tool_policy.
// When sync knows the server's tool catalog, unlisted tools default to ask; YAML entries win.
func mergeToolPolicy(srv ServerPolicy) (map[string]interface{}, []string, error) {
	if len(srv.ToolPolicy) == 0 {
		return nil, nil, nil
	}

	merged := make(map[string]string)
	names, err := knownToolNamesForServer(srv)
	if err != nil {
		return nil, nil, err
	}
	for _, name := range names {
		merged[name] = toolPolicyDefaultUnlisted
	}
	for k, v := range srv.ToolPolicy {
		merged[k] = normalizeToolPolicyValue(v)
	}

	allow, ask, blocked := countToolPolicyActions(merged)
	var warnings []string
	warnings = append(warnings, fmt.Sprintf(
		"[%s] tool_policy: %d allow, %d ask, %d blocked (unlisted default: %s)",
		serverLabel(srv), allow, ask, blocked, toolPolicyDefaultUnlisted,
	))
	return toolPolicyToInterface(merged), warnings, nil
}

func serverLabel(srv ServerPolicy) string {
	if strings.TrimSpace(srv.Name) != "" {
		return srv.Name
	}
	if srv.GoogleService != "" {
		return "google-" + srv.GoogleService
	}
	return "server"
}

func normalizeToolPolicyValue(v string) string {
	switch strings.ToLower(strings.TrimSpace(v)) {
	case "allow", "ask", "blocked":
		return strings.ToLower(strings.TrimSpace(v))
	default:
		return strings.TrimSpace(v)
	}
}

func toolPolicyToInterface(m map[string]string) map[string]interface{} {
	if len(m) == 0 {
		return nil
	}
	out := make(map[string]interface{}, len(m))
	for k, v := range m {
		out[k] = v
	}
	return out
}

func attachToolPolicy(entry map[string]interface{}, srv ServerPolicy) ([]string, error) {
	tp, warnings, err := mergeToolPolicy(srv)
	if err != nil {
		return warnings, err
	}
	if tp != nil {
		entry["toolPolicy"] = tp
	}
	return warnings, nil
}
