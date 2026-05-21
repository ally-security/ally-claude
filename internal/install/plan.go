package install

import (
	"encoding/json"
	"fmt"
	"io"
	"log/slog"

	"github.com/anthropics/claude-3p-helper/internal/policy"
)

type Options struct {
	Activate bool
}

type Plan struct {
	cfg       *policy.Config
	paths     Paths
	doc       map[string]interface{}
	activate  bool
}

func New(c *policy.Config, opts Options) (*Plan, error) {
	paths, err := resolvePaths()
	if err != nil {
		return nil, err
	}
	return &Plan{
		cfg:      c,
		paths:    paths,
		doc:      buildConfigDoc(c),
		activate: opts.Activate,
	}, nil
}

// Print writes a human-readable summary of the plan.
func (p *Plan) Print(w io.Writer) {
	fmt.Fprintln(w, "target paths:")
	fmt.Fprintln(w, "  config:    ", p.paths.ConfigLibrary)
	fmt.Fprintln(w, "  plugins:   ", p.paths.OrgPlugins)
	fmt.Fprintln(w, "  extensions:", p.paths.Extensions)
	fmt.Fprintln(w)

	fmt.Fprintf(w, "config:     %s.json (%d top-level keys)\n", p.cfg.ID, len(p.doc))
	fmt.Fprintf(w, "connectors: %d\n", len(p.cfg.Connectors))
	for _, c := range p.cfg.Connectors {
		fmt.Fprintf(w, "  - %s (%s)\n", c.Name, connectorEndpoint(c))
	}
	fmt.Fprintf(w, "plugins:    %d\n", len(p.cfg.Plugins))
	for _, b := range p.cfg.Plugins {
		fmt.Fprintf(w, "  - %s ← %s\n", b.Name, b.Source)
	}
	fmt.Fprintf(w, "extensions: %d\n", len(p.cfg.Extensions))
	for _, b := range p.cfg.Extensions {
		fmt.Fprintf(w, "  - %s ← %s\n", b.Name, b.Source)
	}
	if p.activate {
		fmt.Fprintln(w, "activate:   yes (will set _meta.activeConfigId =", p.cfg.ID+")")
	} else {
		fmt.Fprintln(w, "activate:   no")
	}
}

func connectorEndpoint(c policy.Connector) string {
	if c.URL != "" {
		return c.URL
	}
	if c.Command != "" {
		return "stdio:" + c.Command
	}
	return "unknown"
}

// Apply writes the config doc and installs plugins/extensions.
func (p *Plan) Apply() error {
	slog.Debug("writing config", "id", p.cfg.ID, "dir", p.paths.ConfigLibrary, "keys", len(p.doc))
	if err := writeConfig(p.paths.ConfigLibrary, p.cfg.ID, p.doc); err != nil {
		return fmt.Errorf("write config: %w", err)
	}
	slog.Info("config written", "id", p.cfg.ID)

	for _, b := range p.cfg.Plugins {
		slog.Debug("installing plugin", "name", b.Name, "source", b.Source)
		if err := installPlugin(p.paths.OrgPlugins, b); err != nil {
			return fmt.Errorf("plugin %s: %w", b.Name, err)
		}
		slog.Info("plugin installed", "name", b.Name)
	}
	for _, b := range p.cfg.Extensions {
		slog.Debug("installing extension", "name", b.Name, "source", b.Source)
		if err := installExtension(p.paths.Extensions, b); err != nil {
			return fmt.Errorf("extension %s: %w", b.Name, err)
		}
		slog.Info("extension installed", "name", b.Name)
	}
	if p.activate {
		if err := activateConfig(p.paths.ConfigLibrary, p.cfg.ID); err != nil {
			return fmt.Errorf("activate: %w", err)
		}
		slog.Info("config activated", "id", p.cfg.ID)
	}
	return nil
}

// Compile-time check that doc is JSON-serializable (catches struct-tag drift).
var _ = json.Marshal
