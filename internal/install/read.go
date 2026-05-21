package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

// LoadedConfig holds the parsed contents of a configLibrary/<id>.json.
type LoadedConfig struct {
	ID                string
	Provider          string
	BedrockRegion     string
	Path              string
	InferenceModels   []interface{}
	Raw               map[string]interface{}
}

// ActiveID returns the activeConfigId from _meta.json, or "" if none is set.
func ActiveID() (string, string, error) {
	paths, err := resolvePaths()
	if err != nil {
		return "", "", err
	}
	metaPath := filepath.Join(paths.ConfigLibrary, "_meta.json")
	data, err := os.ReadFile(metaPath)
	if err != nil {
		if os.IsNotExist(err) {
			return "", paths.ConfigLibrary, nil
		}
		return "", paths.ConfigLibrary, err
	}
	var meta map[string]interface{}
	if err := json.Unmarshal(data, &meta); err != nil {
		return "", paths.ConfigLibrary, fmt.Errorf("parse _meta.json: %w", err)
	}
	id, _ := meta["activeConfigId"].(string)
	return id, paths.ConfigLibrary, nil
}

// LoadConfig reads configLibrary/<id>.json into a LoadedConfig.
func LoadConfig(id string) (*LoadedConfig, error) {
	paths, err := resolvePaths()
	if err != nil {
		return nil, err
	}
	target := filepath.Join(paths.ConfigLibrary, id+".json")
	data, err := os.ReadFile(target)
	if err != nil {
		return nil, fmt.Errorf("read %s: %w", target, err)
	}
	var raw map[string]interface{}
	if err := json.Unmarshal(data, &raw); err != nil {
		return nil, fmt.Errorf("parse %s: %w", target, err)
	}
	c := &LoadedConfig{ID: id, Path: target, Raw: raw}
	if v, ok := raw["inferenceProvider"].(string); ok {
		c.Provider = v
	}
	if v, ok := raw["inferenceBedrockRegion"].(string); ok {
		c.BedrockRegion = v
	}
	if v, ok := raw["inferenceModels"].([]interface{}); ok {
		c.InferenceModels = v
	}
	return c, nil
}

// ListConfigs returns the ids of every <id>.json found in configLibrary.
func ListConfigs() ([]string, error) {
	paths, err := resolvePaths()
	if err != nil {
		return nil, err
	}
	entries, err := os.ReadDir(paths.ConfigLibrary)
	if err != nil {
		if os.IsNotExist(err) {
			return nil, nil
		}
		return nil, err
	}
	var ids []string
	for _, e := range entries {
		name := e.Name()
		if e.IsDir() || !strings.HasSuffix(name, ".json") || name == "_meta.json" {
			continue
		}
		ids = append(ids, strings.TrimSuffix(name, ".json"))
	}
	sort.Strings(ids)
	return ids, nil
}

// FormatModel renders a single inferenceModels entry into a human row.
// Accepts either a bare string or an object with name/labelOverride/supports1m.
func FormatModel(entry interface{}) string {
	switch v := entry.(type) {
	case string:
		return v
	case map[string]interface{}:
		name, _ := v["name"].(string)
		label, _ := v["labelOverride"].(string)
		supports1m, _ := v["supports1m"].(bool)
		out := name
		if label != "" {
			out = fmt.Sprintf("%s  (%s)", name, label)
		}
		if supports1m {
			out += "  [1M]"
		}
		return out
	default:
		return fmt.Sprintf("%v", entry)
	}
}
