package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"slices"
	"strings"
	"text/template"
)

type reachability struct {
	packages map[string]goPackage
	files    map[string]bool
	symbols  map[string]bool
}

type goPackage struct {
	Dir           string
	ImportPath    string
	Name          string
	Standard      bool
	GoFiles       []string
	CgoFiles      []string
	EmbedFiles    []string
	EmbedPatterns []string
}

func (pkg goPackage) symbolPrefix() string {
	if pkg.Name == "main" {
		return "main."
	}
	return pkg.ImportPath + "."
}

func (analysis *reachability) addBuilds(root string, cfg config, paths map[string]bool) error {
	for _, build := range cfg.Builds {
		if build.Main == "" || len(build.GOOS) == 0 || len(build.GOARCH) == 0 {
			return errors.New("release builds must specify main, goos, and goarch")
		}
		tmpl, err := template.New("main").Option("missingkey=error").Parse(build.Main)
		if err != nil {
			return err
		}
		var entrypoint bytes.Buffer
		if err := tmpl.Execute(&entrypoint, cfg); err != nil {
			return err
		}

		for _, goos := range build.GOOS {
			for _, goarch := range build.GOARCH {
				env := append(os.Environ(), build.Env...)
				env = append(env, "GOOS="+goos, "GOARCH="+goarch)
				if err := analysis.addTarget(root, entrypoint.String(), build.Tags, env, paths); err != nil {
					return fmt.Errorf("analyze %s %s/%s: %w", entrypoint.String(), goos, goarch, err)
				}
			}
		}
	}
	return nil
}

func (analysis *reachability) addTarget(root, entrypoint string, tags, env []string, paths map[string]bool) error {
	cmd := exec.Command("go", "list", "-mod=readonly", "-deps", "-json", "-tags="+strings.Join(tags, ","), entrypoint)
	cmd.Env = env
	cmd.Stderr = os.Stderr
	data, err := cmd.Output()
	if err != nil {
		return fmt.Errorf("list dependencies: %w", err)
	}
	if err := analysis.addPackages(root, data, paths); err != nil {
		return err
	}
	return analysis.link(entrypoint, tags, env)
}

func (analysis *reachability) addPackages(root string, data []byte, paths map[string]bool) error {
	decoder := json.NewDecoder(bytes.NewReader(data))
	for {
		var pkg goPackage
		if err := decoder.Decode(&pkg); err != nil {
			if errors.Is(err, io.EOF) {
				return nil
			}
			return err
		}
		if pkg.Standard {
			continue
		}
		dir, err := filepath.Rel(root, pkg.Dir)
		if err != nil {
			return err
		}
		if !filepath.IsLocal(dir) {
			continue
		}
		dir = filepath.ToSlash(dir)
		pkg.EmbedFiles = append(pkg.EmbedFiles, analysis.packages[dir].EmbedFiles...)
		pkg.EmbedPatterns = append(pkg.EmbedPatterns, analysis.packages[dir].EmbedPatterns...)
		slices.Sort(pkg.EmbedFiles)
		pkg.EmbedFiles = slices.Compact(pkg.EmbedFiles)
		analysis.packages[dir] = pkg
		for _, name := range append(pkg.GoFiles, pkg.CgoFiles...) {
			analysis.files[filepath.ToSlash(filepath.Join(dir, name))] = true
		}

		paths[":(top,glob)"+dir+"/*"] = true
		for _, pattern := range pkg.EmbedPatterns {
			pattern = strings.TrimPrefix(pattern, "all:")
			if index := strings.IndexAny(pattern, "*?["); index >= 0 {
				pattern = filepath.Dir(pattern[:index] + "_")
			}
			paths[":(top,literal)"+filepath.ToSlash(filepath.Join(dir, pattern))] = true
		}
	}
}

func (analysis *reachability) link(entrypoint string, tags, env []string) (err error) {
	dir, err := os.MkdirTemp("", "release-reachability-")
	if err != nil {
		return err
	}
	defer func() { err = errors.Join(err, os.RemoveAll(dir)) }()
	cmd := exec.Command("go", "build", "-mod=readonly", "-buildvcs=false", "-gcflags=./...=-l", "-ldflags=-dumpdep", "-tags="+strings.Join(tags, ","), "-o", filepath.Join(dir, "release"), entrypoint)
	cmd.Env = env
	output, err := cmd.CombinedOutput()
	if err != nil {
		return fmt.Errorf("link %s: %s: %w", entrypoint, output, err)
	}

	edges := 0
	for line := range strings.SplitSeq(string(output), "\n") {
		_, symbol, ok := strings.Cut(line, " -> ")
		if !ok {
			continue
		}
		edges++
		symbol, _, _ = strings.Cut(symbol, " <")
		analysis.symbols[withoutTypeArguments(symbol)] = true
	}
	if edges == 0 {
		return errors.New("linker emitted no reachability graph")
	}
	return nil
}

func withoutTypeArguments(name string) string {
	var result strings.Builder
	depth := 0
	for _, character := range name {
		switch character {
		case '[':
			depth++
		case ']':
			depth--
		default:
			if depth == 0 {
				result.WriteRune(character)
			}
		}
	}
	return result.String()
}
