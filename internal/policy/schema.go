package policy

// Config is the top-level YAML schema. Field names map to the keys consumed
// by Claude-3p's configLibrary, except for connectors/plugins/extensions
// which are this tool's own bundling concept.
type Config struct {
	ID   string `yaml:"id"`
	Name string `yaml:"name,omitempty"`

	// Inference provider
	InferenceProvider             string            `yaml:"inferenceProvider"`
	InferenceCustomHeaders        map[string]string `yaml:"inferenceCustomHeaders,omitempty"`
	InferenceAnthropicApiKey      string            `yaml:"inferenceAnthropicApiKey,omitempty"`
	InferenceCredentialHelper     string            `yaml:"inferenceCredentialHelper,omitempty"`
	InferenceCredentialHelperTTL  int               `yaml:"inferenceCredentialHelperTtlSec,omitempty"`

	// Models
	ModelDiscoveryEnabled *bool         `yaml:"modelDiscoveryEnabled,omitempty"`
	InferenceModels       []interface{} `yaml:"inferenceModels,omitempty"`

	// Restrictions
	AllowedWorkspaceFolders  []string          `yaml:"allowedWorkspaceFolders,omitempty"`
	CoworkEgressAllowedHosts []string          `yaml:"coworkEgressAllowedHosts,omitempty"`
	DisabledBuiltinTools     []string          `yaml:"disabledBuiltinTools,omitempty"`
	BuiltinToolPolicy        map[string]string `yaml:"builtinToolPolicy,omitempty"`

	// Telemetry / lifecycle
	DeploymentOrganizationUUID  string `yaml:"deploymentOrganizationUuid,omitempty"`
	DisableEssentialTelemetry   *bool  `yaml:"disableEssentialTelemetry,omitempty"`
	DisableNonessentialTelemetry *bool `yaml:"disableNonessentialTelemetry,omitempty"`
	DisableAutoUpdates          *bool  `yaml:"disableAutoUpdates,omitempty"`

	// OTLP
	OTLPEndpoint string            `yaml:"otlpEndpoint,omitempty"`
	OTLPProtocol string            `yaml:"otlpProtocol,omitempty"`
	OTLPHeaders  map[string]string `yaml:"otlpHeaders,omitempty"`

	// Extensions surface flags
	IsLocalDevMcpEnabled      *bool `yaml:"isLocalDevMcpEnabled,omitempty"`
	IsDesktopExtensionEnabled *bool `yaml:"isDesktopExtensionEnabled,omitempty"`

	// Per-server tool policy overrides for org plugins
	OrgPluginSettings map[string]interface{} `yaml:"orgPluginSettings,omitempty"`

	// Bundles
	Connectors []Connector `yaml:"connectors,omitempty"`
	Plugins    []Bundle    `yaml:"plugins,omitempty"`
	Extensions []Bundle    `yaml:"extensions,omitempty"`
}

// Connector mirrors a managedMcpServers entry.
type Connector struct {
	Name       string            `yaml:"name" json:"name"`
	URL        string            `yaml:"url,omitempty" json:"url,omitempty"`
	Transport  string            `yaml:"transport,omitempty" json:"transport,omitempty"`
	Headers    map[string]string `yaml:"headers,omitempty" json:"headers,omitempty"`
	HeadersHelper string         `yaml:"headersHelper,omitempty" json:"headersHelper,omitempty"`
	OAuth      interface{}       `yaml:"oauth,omitempty" json:"oauth,omitempty"`
	ToolPolicy map[string]string `yaml:"toolPolicy,omitempty" json:"toolPolicy,omitempty"`

	// stdio transport
	Command string            `yaml:"command,omitempty" json:"command,omitempty"`
	Args    []string          `yaml:"args,omitempty" json:"args,omitempty"`
	Env     map[string]string `yaml:"env,omitempty" json:"env,omitempty"`
}

// Bundle is a plugin or extension to fetch and drop on disk.
type Bundle struct {
	Name   string `yaml:"name"`
	Source string `yaml:"source"`
	SHA256 string `yaml:"sha256,omitempty"`
}
