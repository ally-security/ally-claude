package main

import (
	"flag"
	"fmt"
	"log/slog"
	"os"

	"github.com/anthropics/claude-3p-helper/internal/install"
	"github.com/anthropics/claude-3p-helper/internal/logging"
)

func runModels(args []string) error {
	fs := flag.NewFlagSet("models", flag.ExitOnError)
	configID := fs.String("config", "", "inspect a specific configLibrary id (default: active)")
	all := fs.Bool("all", false, "list all synced configs and their declared models")
	verbose := fs.Bool("verbose", false, "emit debug-level logs to stderr")
	_ = fs.Parse(args)
	logging.Setup(*verbose)
	slog.Debug("models invoked", "configID", *configID, "all", *all)

	if *all {
		ids, err := install.ListConfigs()
		if err != nil {
			return err
		}
		if len(ids) == 0 {
			fmt.Println("(no configs found in configLibrary)")
			return nil
		}
		for i, id := range ids {
			if i > 0 {
				fmt.Println()
			}
			if err := printConfigModels(id); err != nil {
				return err
			}
		}
		return nil
	}

	id := *configID
	if id == "" {
		active, dir, err := install.ActiveID()
		if err != nil {
			return err
		}
		if active == "" {
			return fmt.Errorf("no active config in %s — run `sync` first, or pass --config <id>", dir)
		}
		id = active
	}
	return printConfigModels(id)
}

func printConfigModels(id string) error {
	cfg, err := install.LoadConfig(id)
	if err != nil {
		return err
	}
	fmt.Fprintf(os.Stdout, "config: %s\n", cfg.ID)
	fmt.Fprintf(os.Stdout, "path:   %s\n", cfg.Path)
	if cfg.Provider != "" {
		fmt.Fprintf(os.Stdout, "provider: %s", cfg.Provider)
		if cfg.Provider == "bedrock" && cfg.BedrockRegion != "" {
			fmt.Fprintf(os.Stdout, " (%s)", cfg.BedrockRegion)
		}
		fmt.Fprintln(os.Stdout)
	}
	fmt.Fprintf(os.Stdout, "models: %d\n", len(cfg.InferenceModels))
	if len(cfg.InferenceModels) == 0 {
		fmt.Fprintln(os.Stdout, "  (none declared — auto-discovery may be in effect)")
		return nil
	}
	for _, m := range cfg.InferenceModels {
		fmt.Fprintf(os.Stdout, "  - %s\n", install.FormatModel(m))
	}
	return nil
}
