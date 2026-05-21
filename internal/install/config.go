package install

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/anthropics/claude-3p-helper/internal/policy"
)

// buildConfigDoc projects the policy YAML into the JSON shape stored in
// configLibrary/<id>.json. The configLibrary uses native JSON types
// (booleans, arrays, objects) — string-encoding is only required for the
// MDM plist/registry path, which this tool doesn't touch.
func buildConfigDoc(c *policy.Config) map[string]interface{} {
	doc := map[string]interface{}{
		"id":                c.ID,
		"inferenceProvider": c.InferenceProvider,
	}
	put := func(k string, v interface{}) {
		if v != nil {
			doc[k] = v
		}
	}
	putStr := func(k, v string) {
		if v != "" {
			doc[k] = v
		}
	}
	putMap := func(k string, m map[string]string) {
		if len(m) > 0 {
			doc[k] = m
		}
	}
	putSlice := func(k string, s []string) {
		if len(s) > 0 {
			doc[k] = s
		}
	}

	if c.Name != "" {
		doc["name"] = c.Name
	}
	putMap("inferenceCustomHeaders", c.InferenceCustomHeaders)
	putStr("inferenceAnthropicApiKey", c.InferenceAnthropicApiKey)
	putStr("inferenceCredentialHelper", c.InferenceCredentialHelper)
	if c.InferenceCredentialHelperTTL > 0 {
		doc["inferenceCredentialHelperTtlSec"] = c.InferenceCredentialHelperTTL
	}
	putStr("inferenceBedrockRegion", c.InferenceBedrockRegion)
	putStr("inferenceBedrockBearerToken", c.InferenceBedrockBearerToken)
	putStr("inferenceBedrockProfile", c.InferenceBedrockProfile)
	putStr("inferenceBedrockSsoStartUrl", c.InferenceBedrockSsoStartUrl)
	putStr("inferenceBedrockSsoRegion", c.InferenceBedrockSsoRegion)
	putStr("inferenceBedrockSsoAccountId", c.InferenceBedrockSsoAccountId)
	putStr("inferenceBedrockSsoRoleName", c.InferenceBedrockSsoRoleName)
	put("modelDiscoveryEnabled", boolPtr(c.ModelDiscoveryEnabled))
	if len(c.InferenceModels) > 0 {
		doc["inferenceModels"] = c.InferenceModels
	}

	putSlice("allowedWorkspaceFolders", c.AllowedWorkspaceFolders)
	putSlice("coworkEgressAllowedHosts", c.CoworkEgressAllowedHosts)
	putSlice("disabledBuiltinTools", c.DisabledBuiltinTools)
	putMap("builtinToolPolicy", c.BuiltinToolPolicy)

	putStr("deploymentOrganizationUuid", c.DeploymentOrganizationUUID)
	put("disableEssentialTelemetry", boolPtr(c.DisableEssentialTelemetry))
	put("disableNonessentialTelemetry", boolPtr(c.DisableNonessentialTelemetry))
	put("disableAutoUpdates", boolPtr(c.DisableAutoUpdates))

	putStr("otlpEndpoint", c.OTLPEndpoint)
	putStr("otlpProtocol", c.OTLPProtocol)
	putMap("otlpHeaders", c.OTLPHeaders)

	put("isLocalDevMcpEnabled", boolPtr(c.IsLocalDevMcpEnabled))
	put("isDesktopExtensionEnabled", boolPtr(c.IsDesktopExtensionEnabled))

	if len(c.OrgPluginSettings) > 0 {
		doc["orgPluginSettings"] = c.OrgPluginSettings
	}
	if len(c.Connectors) > 0 {
		doc["managedMcpServers"] = c.Connectors
	}
	return doc
}

func boolPtr(p *bool) interface{} {
	if p == nil {
		return nil
	}
	return *p
}

func writeConfig(dir, id string, doc map[string]interface{}) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	target := filepath.Join(dir, id+".json")
	return writeJSONAtomic(target, doc)
}

func activateConfig(dir, id string) error {
	if err := os.MkdirAll(dir, 0o755); err != nil {
		return err
	}
	meta := filepath.Join(dir, "_meta.json")

	current := map[string]interface{}{}
	if data, err := os.ReadFile(meta); err == nil {
		_ = json.Unmarshal(data, &current)
	}
	current["activeConfigId"] = id
	return writeJSONAtomic(meta, current)
}

func writeJSONAtomic(target string, v interface{}) error {
	data, err := json.MarshalIndent(v, "", "  ")
	if err != nil {
		return err
	}
	tmp := target + ".tmp"
	if err := os.WriteFile(tmp, data, 0o644); err != nil {
		return err
	}
	if err := os.Rename(tmp, target); err != nil {
		return fmt.Errorf("atomic rename to %s: %w", target, err)
	}
	return nil
}
