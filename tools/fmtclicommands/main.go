package main

import (
	"bytes"
	"errors"
	"flag"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
)

var errFormatting = errors.New("CLI command declarations need formatting")

func main() {
	if err := run(os.Args[1:], os.Stderr); err != nil {
		if !errors.Is(err, errFormatting) && !errors.Is(err, flag.ErrHelp) {
			fmt.Fprintln(os.Stderr, err)
		}
		if !errors.Is(err, flag.ErrHelp) {
			os.Exit(1)
		}
	}
}

func run(args []string, stderr io.Writer) error {
	flags := flag.NewFlagSet("fmtclicommands", flag.ContinueOnError)
	flags.SetOutput(stderr)
	check := flags.Bool("check", false, "check formatting without writing (default)")
	write := flags.Bool("write", false, "format CLI command declarations in place")
	flags.Usage = func() {
		fmt.Fprintln(stderr, "Usage: fmtclicommands [--check | --write] [files or directories...] (default: cmd/api; skips tests, testdata, and testutil)")
		flags.PrintDefaults()
	}
	if err := flags.Parse(args); err != nil {
		return err
	}
	if *check && *write {
		return errors.New("choose either --check or --write, not both")
	}
	paths := flags.Args()
	if len(paths) == 0 {
		paths = []string{"cmd/api"}
	}

	violations := 0
	for _, root := range paths {
		err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, walkErr error) error {
			if walkErr != nil {
				return walkErr
			}
			if entry.IsDir() {
				if entry.Name() == "testdata" || entry.Name() == "testutil" || strings.HasPrefix(entry.Name(), ".") && path != root {
					return filepath.SkipDir
				}
				return nil
			}
			if entry.Type()&os.ModeSymlink != 0 || !strings.HasSuffix(path, ".go") || strings.HasSuffix(path, "_test.go") {
				return nil
			}
			changed, err := processFile(path, *write)
			if err != nil {
				return err
			}
			if changed && !*write {
				fmt.Fprintf(stderr, "%s: CLI fields and Flags/Examples/Commands elements must use separate lines; run mise run fmt-cli-commands -- %s\n", path, path)
				violations++
			}
			return nil
		})
		if err != nil {
			return err
		}
	}
	if violations > 0 {
		return errFormatting
	}
	return nil
}

func processFile(path string, write bool) (bool, error) {
	source, err := os.ReadFile(path)
	if err != nil {
		return false, fmt.Errorf("read %s: %w", path, err)
	}
	formatted, err := formatCommands(path, source)
	if err != nil {
		return false, fmt.Errorf("format %s: %w", path, err)
	}
	if bytes.Equal(source, formatted) {
		return false, nil
	}
	if write {
		if err := os.WriteFile(path, formatted, 0o644); err != nil {
			return false, fmt.Errorf("write %s: %w", path, err)
		}
	}
	return true, nil
}
