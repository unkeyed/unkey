package main

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"os"
	"slices"
)

func main() {
	if len(os.Args) != 2 {
		fmt.Fprintln(os.Stderr, "usage: releaseconfig <goreleaser.yaml>")
		os.Exit(1)
	}
	if err := run(os.Args[1], os.Stdout); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func run(filename string, output io.Writer) error {
	paths := map[string]bool{"go.mod": true, "go.sum": true}
	cfg, err := readConfig(filename, paths)
	if err != nil {
		return err
	}
	if len(cfg.Builds) == 0 {
		return errors.New("release config has no builds")
	}

	root, err := os.Getwd()
	if err != nil {
		return err
	}
	analysis := reachability{
		packages: map[string]goPackage{},
		files:    map[string]bool{},
		symbols:  map[string]bool{},
	}
	if err := analysis.addBuilds(root, cfg, paths); err != nil {
		return err
	}
	for _, docker := range cfg.Dockers {
		if docker.Dockerfile != "" {
			paths[docker.Dockerfile] = true
		}
	}

	changelog, err := analysis.changelog(cfg, slices.Sorted(maps.Keys(paths)))
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(configOverlay{
		Version:   2,
		Includes:  []configInclude{{FromFile: configFile{Path: filename}}},
		Changelog: changelog,
	})
}
