// Command sqlcretry adds bounded retries to safe sqlc-generated MySQL reads.
package main

import (
	"bytes"
	"flag"
	"fmt"
	"os"
	"path/filepath"
)

func main() {
	check := flag.Bool("check", false, "check generated retries without writing files")
	flag.Parse()
	dirs := flag.Args()
	if len(dirs) == 0 {
		dirs = []string{"."}
	}
	for _, dir := range dirs {
		if err := run(dir, *check); err != nil {
			fmt.Fprintln(os.Stderr, err)
			os.Exit(1)
		}
	}
}

func run(dir string, check bool) error {
	paths, err := filepath.Glob(filepath.Join(dir, "*.sql_generated.go"))
	if err != nil {
		return err
	}
	if len(paths) == 0 {
		return fmt.Errorf("%s: no sqlc-generated query files found", dir)
	}
	for _, path := range paths {
		source, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		generated, err := rewrite(source)
		if err != nil {
			return fmt.Errorf("%s: %w", path, err)
		}
		if bytes.Equal(source, generated) {
			continue
		}
		if check {
			return fmt.Errorf("%s: read retries are missing; run go generate for this package", path)
		}
		if err := os.WriteFile(path, generated, 0o644); err != nil {
			return err
		}
	}
	return nil
}
