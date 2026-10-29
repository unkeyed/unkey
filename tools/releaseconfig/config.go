package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"gopkg.in/yaml.v3"
)

type config struct {
	ProjectName string `yaml:"project_name"`
	Monorepo    struct {
		TagPrefix string `yaml:"tag_prefix"`
	} `yaml:"monorepo"`
	Changelog struct {
		Filters changelogFilters `yaml:"filters"`
	} `yaml:"changelog"`
	Includes []configInclude `yaml:"includes"`
	Builds   []buildConfig   `yaml:"builds"`
	Dockers  []struct {
		Dockerfile string `yaml:"dockerfile"`
	} `yaml:"dockers_v2"`
}

type configInclude struct {
	FromFile configFile `yaml:"from_file" json:"from_file"`
}

type configFile struct {
	Path string `yaml:"path" json:"path"`
}

type buildConfig struct {
	Main   string   `yaml:"main"`
	Env    []string `yaml:"env"`
	GOOS   []string `yaml:"goos"`
	GOARCH []string `yaml:"goarch"`
	Tags   []string `yaml:"tags"`
}

type changelogFilters struct {
	Exclude []string `yaml:"exclude" json:"exclude"`
	Include []string `yaml:"include" json:"include,omitempty"`
}

type configOverlay struct {
	Changelog changelogConfig `json:"changelog"`
	Includes  []configInclude `json:"includes"`
	Version   int             `json:"version"`
}

type changelogConfig struct {
	Filters *changelogFilters `json:"filters,omitempty"`
	Format  string            `json:"format"`
	Paths   []string          `json:"paths"`
	Use     string            `json:"use"`
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
		cfg.inherit(base)
	}
	return cfg, nil
}

func (cfg *config) inherit(base config) {
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
