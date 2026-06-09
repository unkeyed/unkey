// Command release tags one or more Unkey services and pushes the tags to origin.
// Pushing a `<service>/vx.y.z` tag drives CI to build the image, push it to
// GHCR, and cut a GitHub release. Versions are auto-numbered from existing tags
// unless pinned; --bump selects the stable increment and --rc/--pre cut a
// pre-release. See docs/engineering/contributing/tooling/releases.mdx.
package main

import (
	"context"
	"errors"
	"fmt"
	"os"

	"github.com/unkeyed/unkey/pkg/cli"
)

func main() {
	if err := cmd().Run(context.Background(), os.Args); err != nil {
		fmt.Fprintln(os.Stderr, "error:", err)
		os.Exit(1)
	}
}

// cmd builds the release command, its flags, and validation.
func cmd() *cli.Command {
	return &cli.Command{
		Name:        "release",
		Usage:       "Tag and push Unkey service releases",
		Description: "Tag one or more services and push the tags to origin. Versions are auto-numbered from existing tags unless pinned explicitly.",
		Examples: []string{
			"release api vault                    # auto patch-bump each service",
			"release --bump minor api             # auto minor-bump",
			"release --rc api                     # next release candidate (-rc.N)",
			"release --bump minor --rc api vault  # next minor-version rc",
			"release --version v1.2.3 api vault   # pin an explicit version",
			"release api@v1.2.3 vault@v0.4.0-rc.1 # per-service explicit versions",
			"release --dry-run --rc api           # preview only",
		},
		AcceptsArgs: true,
		Version:     "",
		Aliases:     []string{},
		Commands:    []*cli.Command{},
		Flags: []cli.Flag{
			cli.String("version", "Pin an exact version (vX.Y.Z[-pre]) for all bare service args", cli.Validate(validSemver)),
			cli.Enum("bump", "Auto-bump kind when no version is given", bumpKinds(), cli.Default(string(bumpPatch))),
			cli.String("pre", "Auto-number the next pre-release with this label (rc, beta, alpha, ...)"),
			cli.Bool("rc", "Shorthand for --pre rc"),
			cli.Bool("dry-run", "Print the plan without creating or pushing tags"),
			cli.Bool("yes", "Skip the confirmation prompt"),
			cli.Bool("no-log", "Don't print the commits/PRs merged since the last release"),
			cli.Bool("no-fetch", "Skip fetching origin tags before numbering (offline)"),
			cli.Bool("no-verify", "Skip git safety checks (on-main, tag-exists)"),
		},
		Action: action,
	}
}

// action translates parsed flags into options and runs the release.
func action(_ context.Context, cmd *cli.Command) error {
	if cmd.FlagIsSet("version") && (cmd.FlagIsSet("bump") || cmd.FlagIsSet("rc") || cmd.FlagIsSet("pre")) {
		return errors.New("--version cannot be combined with --bump/--rc/--pre (it pins an exact version)")
	}
	if cmd.Bool("rc") && cmd.FlagIsSet("pre") {
		return errors.New("--rc and --pre cannot be combined")
	}

	services := cmd.Args()
	if len(services) == 0 {
		return errors.New("no services specified (e.g. `release api vault` or `release --rc api`)")
	}

	pre := cmd.String("pre")
	if cmd.Bool("rc") {
		pre = "rc"
	}

	return release(options{
		version:  cmd.String("version"),
		bump:     bumpKind(cmd.Enum("bump")),
		pre:      pre,
		services: services,
		dryRun:   cmd.Bool("dry-run"),
		yes:      cmd.Bool("yes"),
		noFetch:  cmd.Bool("no-fetch"),
		noLog:    cmd.Bool("no-log"),
		noVerify: cmd.Bool("no-verify"),
	})
}
