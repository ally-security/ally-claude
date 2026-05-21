package main

import (
	"flag"
	"fmt"
	"os"

	"github.com/anthropics/claude-3p-helper/internal/install"
	"github.com/anthropics/claude-3p-helper/internal/policy"
	"github.com/anthropics/claude-3p-helper/internal/source"
	"github.com/anthropics/claude-3p-helper/internal/version"
)

func main() {
	if len(os.Args) < 2 {
		usage()
		os.Exit(2)
	}
	switch os.Args[1] {
	case "sync":
		if err := runSync(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "sync:", err)
			os.Exit(1)
		}
	case "models":
		if err := runModels(os.Args[2:]); err != nil {
			fmt.Fprintln(os.Stderr, "models:", err)
			os.Exit(1)
		}
	case "version", "--version", "-v":
		fmt.Println(version.String())
	case "help", "--help", "-h":
		usage()
	default:
		fmt.Fprintln(os.Stderr, "unknown command:", os.Args[1])
		usage()
		os.Exit(2)
	}
}

func usage() {
	fmt.Fprintln(os.Stderr, `claude-3p-helper — sync MDM-style policy to Claude-3p

Usage:
  claude-3p-helper sync [flags] <user>/<repo>/<path>
  claude-3p-helper models [flags]
  claude-3p-helper version

Sync flags:
  --branch string   git branch when fetching from GitHub (default "main")
  --dry-run         print planned actions, don't write
  --no-activate     don't mark the synced config as active

Models flags:
  --config string   inspect a specific configLibrary id (default: active)
  --all             list all synced configs and their declared models`)
}

func runSync(args []string) error {
	fs := flag.NewFlagSet("sync", flag.ExitOnError)
	branch := fs.String("branch", "main", "git branch when fetching from GitHub")
	dryRun := fs.Bool("dry-run", false, "print planned actions, don't write")
	noActivate := fs.Bool("no-activate", false, "don't mark the synced config as active")
	_ = fs.Parse(args)

	if fs.NArg() != 1 {
		return fmt.Errorf("expected exactly one <user>/<repo>/<path> argument")
	}

	data, origin, err := source.Resolve(fs.Arg(0), *branch)
	if err != nil {
		return err
	}
	fmt.Println("loaded policy from:", origin)

	cfg, err := policy.Load(data)
	if err != nil {
		return err
	}

	plan, err := install.New(cfg, install.Options{Activate: !*noActivate})
	if err != nil {
		return err
	}

	if *dryRun {
		plan.Print(os.Stdout)
		fmt.Println("\n(dry-run, no changes written)")
		return nil
	}

	if err := plan.Apply(); err != nil {
		return err
	}
	plan.Print(os.Stdout)
	fmt.Println("\nfully quit and reopen Claude-3p for changes to take effect")
	return nil
}
