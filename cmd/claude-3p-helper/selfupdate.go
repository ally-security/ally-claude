package main

import (
	"flag"
	"fmt"
	"log/slog"
	"net/http"
	"os"

	"github.com/anthropics/claude-3p-helper/internal/logging"
	"github.com/anthropics/claude-3p-helper/internal/selfupdate"
	"github.com/anthropics/claude-3p-helper/internal/version"
)

// selfUpdateClient is the HTTP client used by self-update; tests override it.
var selfUpdateClient *http.Client

func runSelfUpdate(args []string) error {
	fs := flag.NewFlagSet("self-update", flag.ExitOnError)
	dryRun := fs.Bool("check", false, "check for updates without replacing the binary")
	verbose := fs.Bool("verbose", false, "emit debug-level logs to stderr")
	_ = fs.Parse(args)
	logging.Setup(*verbose)

	target, err := os.Executable()
	if err != nil {
		return fmt.Errorf("locate own executable: %w", err)
	}
	slog.Info("self-update starting", "current", version.Version, "repo", version.Repo, "target", target, "check", *dryRun)

	res, err := selfupdate.Run(selfupdate.Options{
		Repo:       version.Repo,
		Current:    version.Version,
		HTTPClient: selfUpdateClient,
	}, target, *dryRun)
	if err != nil {
		return err
	}

	if res.UpToDate {
		fmt.Printf("already up to date (%s)\n", res.Latest)
		slog.Info("up to date", "version", res.Latest)
		return nil
	}
	if *dryRun {
		fmt.Printf("update available: %s -> %s (asset: %s)\n", res.Current, res.Latest, res.Asset)
		slog.Info("update available", "current", res.Current, "latest", res.Latest, "asset", res.Asset)
		return nil
	}
	fmt.Printf("updated %s -> %s (%s)\n", res.Current, res.Latest, res.Path)
	slog.Info("self-update complete", "from", res.Current, "to", res.Latest, "path", res.Path)
	return nil
}
