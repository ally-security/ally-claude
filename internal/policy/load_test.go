package policy

import (
	"strings"
	"testing"
)

func TestLoadMinimalAnthropic(t *testing.T) {
	c, err := Load([]byte(`
id: x
inferenceProvider: anthropic
`))
	if err != nil {
		t.Fatalf("Load: %v", err)
	}
	if c.ID != "x" || c.InferenceProvider != "anthropic" {
		t.Errorf("fields not parsed: %+v", c)
	}
}

func TestLoadBedrockHappyPaths(t *testing.T) {
	cases := map[string]string{
		"bearer":  "inferenceBedrockBearerToken: tok\n",
		"profile": "inferenceBedrockProfile: dev\n",
		"sso":     "inferenceBedrockSsoStartUrl: https://example/sso\n",
		"helper":  "inferenceCredentialHelper: /usr/local/bin/helper\n",
	}
	for name, extra := range cases {
		t.Run(name, func(t *testing.T) {
			y := "id: x\ninferenceProvider: bedrock\ninferenceBedrockRegion: us-west-2\n" + extra
			if _, err := Load([]byte(y)); err != nil {
				t.Fatalf("expected ok, got %v", err)
			}
		})
	}
}

func TestLoadErrors(t *testing.T) {
	cases := map[string]struct {
		yaml    string
		wantSub string
	}{
		"no-id":              {`inferenceProvider: anthropic`, "id is required"},
		"no-provider":        {`id: x`, "inferenceProvider is required"},
		"bedrock-no-region":  {"id: x\ninferenceProvider: bedrock\n", "inferenceBedrockRegion"},
		"bedrock-no-auth":    {"id: x\ninferenceProvider: bedrock\ninferenceBedrockRegion: us\n", "provide one of"},
		"bedrock-multi-auth": {"id: x\ninferenceProvider: bedrock\ninferenceBedrockRegion: us\ninferenceBedrockBearerToken: t\ninferenceBedrockProfile: p\n", "only one auth path"},
		"plugin-no-name":     {"id: x\ninferenceProvider: anthropic\nplugins:\n  - source: x\n", "name and source are required"},
		"plugin-no-source":   {"id: x\ninferenceProvider: anthropic\nplugins:\n  - name: x\n", "name and source are required"},
		"ext-no-name":        {"id: x\ninferenceProvider: anthropic\nextensions:\n  - source: x\n", "name and source are required"},
		"connector-no-name":  {"id: x\ninferenceProvider: anthropic\nconnectors:\n  - url: y\n", "name is required"},
		"bad-yaml":           {"id: [", "parse yaml"},
	}
	for name, tc := range cases {
		t.Run(name, func(t *testing.T) {
			_, err := Load([]byte(tc.yaml))
			if err == nil {
				t.Fatalf("expected error containing %q, got nil", tc.wantSub)
			}
			if !strings.Contains(err.Error(), tc.wantSub) {
				t.Errorf("err = %v, want substring %q", err, tc.wantSub)
			}
		})
	}
}
