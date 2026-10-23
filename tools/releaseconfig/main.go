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

	"gopkg.in/yaml.v3"
)

type config struct {
	ProjectName string `yaml:"project_name"`
	Monorepo    struct {
		TagPrefix string `yaml:"tag_prefix"`
	} `yaml:"monorepo"`
	Changelog struct {
		Filters struct {
			Exclude []string `yaml:"exclude"`
			Include []string `yaml:"include"`
		} `yaml:"filters"`
	} `yaml:"changelog"`
	Includes []struct {
		FromFile struct {
			Path string `yaml:"path"`
		} `yaml:"from_file"`
	} `yaml:"includes"`
	Builds []struct {
		Main   string   `yaml:"main"`
		Env    []string `yaml:"env"`
		Goos   []string `yaml:"goos"`
		Goarch []string `yaml:"goarch"`
		Tags   []string `yaml:"tags"`
	} `yaml:"builds"`
	Dockers []struct {
		Dockerfile string `yaml:"dockerfile"`
	} `yaml:"dockers_v2"`
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
	if err := cfg.addDependencies(root, paths, &analysis); err != nil {
		return err
	}
	for _, docker := range cfg.Dockers {
		if docker.Dockerfile != "" {
			paths[docker.Dockerfile] = true
		}
	}
	ordered := make([]string, 0, len(paths))
	for path := range paths {
		ordered = append(ordered, path)
	}
	slices.Sort(ordered)
	changelog, err := analysis.changelog(cfg, ordered)
	if err != nil {
		return err
	}
	return json.NewEncoder(output).Encode(map[string]interface{}{
		"version": 2,
		"includes": []interface{}{
			map[string]interface{}{"from_file": map[string]string{"path": filename}},
		},
		"changelog": changelog,
	})
}

func (cfg config) addDependencies(root string, paths map[string]bool, analysis *reachability) error {
	for _, build := range cfg.Builds {
		if build.Main == "" || len(build.Goos) == 0 || len(build.Goarch) == 0 {
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
		for _, goos := range build.Goos {
			for _, goarch := range build.Goarch {
				cmd := exec.Command("go", "list", "-mod=readonly", "-deps", "-json", "-tags="+strings.Join(build.Tags, ","), entrypoint.String())
				cmd.Env = append(os.Environ(), build.Env...)
				cmd.Env = append(cmd.Env, "GOOS="+goos, "GOARCH="+goarch)
				cmd.Stderr = os.Stderr
				data, err := cmd.Output()
				if err != nil {
					return fmt.Errorf("list dependencies for %s %s/%s: %w", entrypoint.String(), goos, goarch, err)
				}
				if err := addPackages(root, data, paths, analysis); err != nil {
					return err
				}
				if err := analysis.link(entrypoint.String(), build.Tags, cmd.Env); err != nil {
					return err
				}
			}
		}
	}
	return nil
}

func readConfig(filename string, paths map[string]bool) (config, error) {
	var cfg config
	filename = filepath.Clean(filename)
	if paths[filename] {
		return cfg, fmt.Errorf("duplicate or cyclic config include: %s", filename)
	}
	paths[filename] = true
	data, err := os.ReadFile(filename)
	if err != nil {
		return cfg, err
	}
	if err := yaml.Unmarshal(data, &cfg); err != nil {
		return cfg, err
	}
	for _, include := range cfg.Includes {
		if include.FromFile.Path == "" {
			return cfg, errors.New("release config requires local file includes")
		}
		base, err := readConfig(include.FromFile.Path, paths)
		if err != nil {
			return cfg, err
		}
		if cfg.ProjectName == "" {
			cfg.ProjectName = base.ProjectName
		}
		if cfg.Monorepo.TagPrefix == "" {
			cfg.Monorepo = base.Monorepo
		}
		if cfg.Changelog.Filters.Exclude == nil && cfg.Changelog.Filters.Include == nil {
			cfg.Changelog = base.Changelog
		}
		if len(cfg.Builds) == 0 {
			cfg.Builds = base.Builds
		}
		if len(cfg.Dockers) == 0 {
			cfg.Dockers = base.Dockers
		}
	}
	return cfg, nil
}

func addPackages(root string, data []byte, paths map[string]bool, analysis *reachability) error {
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
