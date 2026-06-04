package claude3p

import (
	"bufio"
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"regexp"
	"strings"
)

var toolNameRE = regexp.MustCompile(`"name"\s*:\s*"([^"]+)"`)

// IsReadOnlyToolName heuristically classifies MCP tools that are safe to pre-allow.
func IsReadOnlyToolName(name string) bool {
	n := strings.ToLower(strings.TrimSpace(name))
	if n == "" {
		return false
	}
	readonlyPrefixes := []string{
		"get_", "list_", "search_", "read_", "query_", "fetch_", "retrieve_",
		"download_", "show_", "describe_", "lookup_", "find_", "view_",
		"debug_", "docs-", "docs_", "execute-sql", "hogql-schema", "read-data",
		"entity-search", "property-definitions", "event-definitions",
	}
	for _, p := range readonlyPrefixes {
		if strings.HasPrefix(n, p) {
			return true
		}
	}
	readonlyContains := []string{
		"-get", "-get-all", "-list", "-search", "-query", "-retrieve",
		"-read", "-stats", "-simulate", "-count", "-schema", "-run-history",
		"insight-query", "switch-project", "switch-organization", "projects-get",
		"tool_guidance", "submit_feedback", "show_feedback",
	}
	for _, s := range readonlyContains {
		if strings.Contains(n, s) {
			return true
		}
	}
	// Docs search tools (e.g. ally_docs_public)
	if strings.Contains(n, "search_") || strings.Contains(n, "query_docs") {
		return true
	}
	return false
}

func mergeToolPolicy(explicit map[string]string, toolNames []string, autoReadonly bool) map[string]string {
	out := make(map[string]string)
	for k, v := range explicit {
		if k = strings.TrimSpace(k); k != "" {
			out[k] = strings.TrimSpace(v)
		}
	}
	if !autoReadonly {
		return out
	}
	for _, name := range toolNames {
		name = strings.TrimSpace(name)
		if name == "" {
			continue
		}
		if _, ok := out[name]; ok {
			continue
		}
		if IsReadOnlyToolName(name) {
			out[name] = "allow"
		} else {
			out[name] = "ask"
		}
	}
	return out
}

func fetchMCPToolNames(mcpURL string, headers map[string]string) ([]string, error) {
	body := []byte(`{"jsonrpc":"2.0","id":1,"method":"tools/list","params":{}}`)
	req, err := http.NewRequest(http.MethodPost, mcpURL, bytes.NewReader(body))
	if err != nil {
		return nil, err
	}
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json, text/event-stream")
	for k, v := range headers {
		req.Header.Set(k, v)
	}
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return nil, err
	}
	defer resp.Body.Close()
	data, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 400 {
		return nil, fmt.Errorf("tools/list HTTP %d", resp.StatusCode)
	}
	var names []string
	seen := make(map[string]struct{})
	sc := bufio.NewScanner(bytes.NewReader(data))
	for sc.Scan() {
		for _, m := range toolNameRE.FindAllStringSubmatch(sc.Text(), -1) {
			if len(m) < 2 {
				continue
			}
			if _, ok := seen[m[1]]; ok {
				continue
			}
			seen[m[1]] = struct{}{}
			names = append(names, m[1])
		}
	}
	if len(names) == 0 {
		for _, m := range toolNameRE.FindAllStringSubmatch(string(data), -1) {
			if len(m) >= 2 {
				if _, ok := seen[m[1]]; ok {
					continue
				}
				seen[m[1]] = struct{}{}
				names = append(names, m[1])
			}
		}
	}
	if len(names) == 0 {
		return nil, fmt.Errorf("no tools found in tools/list response")
	}
	return names, nil
}

func authHeadersFromHelper(helperPath string) (map[string]string, error) {
	if strings.TrimSpace(helperPath) == "" {
		return nil, nil
	}
	out, err := runHelperJSON(helperPath)
	if err != nil {
		return nil, err
	}
	var headers map[string]string
	if err := json.Unmarshal(out, &headers); err != nil {
		return nil, err
	}
	return headers, nil
}
