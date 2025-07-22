package cli

import (
	"fmt"
	"regexp"
	"strings"
	"text/template"
)

// FrontMatter holds metadata for the MDX file
type FrontMatter struct {
	Title       string
	Description string
}

// GenerateMDX creates Fuma docs MDX from any command metadata with frontmatter
func (c *Command) GenerateMDX(frontMatter *FrontMatter) (string, error) {
	data := c.extractMDXData(frontMatter)

	tmpl, err := template.New("mdx").Funcs(template.FuncMap{
		"join":      strings.Join,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"title":     strings.Title,
		"lower":     strings.ToLower,
		"hasItems":  func(items any) bool { return c.hasItems(items) },
		"eq":        func(a, b string) bool { return a == b },
	}).Parse(mdxTemplate)
	if err != nil {
		return "", fmt.Errorf("failed to parse MDX template: %w", err)
	}

	var result strings.Builder
	if err := tmpl.Execute(&result, data); err != nil {
		return "", fmt.Errorf("failed to execute MDX template: %w", err)
	}

	// If this command has subcommands, append their documentation
	if len(c.Commands) > 0 {
		for _, subCmd := range c.Commands {
			subFrontMatter := &FrontMatter{
				Title:       fmt.Sprintf("%s %s", strings.Title(c.Name), strings.Title(subCmd.Name)),
				Description: subCmd.Usage,
			}

			subMDX, err := subCmd.GenerateMDX(subFrontMatter)
			if err != nil {
				return "", fmt.Errorf("failed to generate MDX for subcommand %s: %w", subCmd.Name, err)
			}

			// Remove frontmatter from subcommand MDX and append
			subMDXContent := c.removeFrontmatter(subMDX)
			result.WriteString("\n\n---\n\n")
			result.WriteString(subMDXContent)
		}
	}

	return result.String(), nil
}

// removeFrontmatter strips the frontmatter section from MDX content
func (c *Command) removeFrontmatter(mdx string) string {
	lines := strings.Split(mdx, "\n")
	if len(lines) == 0 || lines[0] != "---" {
		return mdx
	}

	// Find the closing ---
	for i := 1; i < len(lines); i++ {
		if lines[i] == "---" {
			// Return everything after the closing frontmatter
			return strings.Join(lines[i+1:], "\n")
		}
	}

	return mdx
}

// MDXData holds structured command data for template generation
type MDXData struct {
	Name        string
	Usage       string
	Description string
	HasSubcmds  bool
	Subcommands []MDXSubcommand
	Examples    []MDXExample
	Flags       []MDXFlag
	EnvVars     []MDXEnvVar
	CommandType string
	FrontMatter *FrontMatter
}

type MDXSubcommand struct {
	Name        string
	Usage       string
	Description string
	Aliases     []string
}

type MDXExample struct {
	Title   string
	Command string
	Comment string
}

type MDXFlag struct {
	Name        string
	Description string
	Type        string
	Default     string
	EnvVar      string
	Required    bool
	Category    string
	Icon        string
}

type MDXEnvVar struct {
	Name        string
	Description string
	Type        string
}

// extractMDXData parses command metadata into structured data
func (c *Command) extractMDXData(frontMatter *FrontMatter) MDXData {
	return MDXData{
		Name:        c.Name,
		Usage:       c.Usage,
		Description: c.Description,
		HasSubcmds:  len(c.Commands) > 0,
		Subcommands: c.extractSubcommands(),
		Examples:    c.extractAllExamples(),
		Flags:       c.extractAllFlags(),
		EnvVars:     c.extractEnvVariables(),
		CommandType: c.determineCommandType(),
		FrontMatter: frontMatter,
	}
}

// extractSubcommands gets subcommand information
func (c *Command) extractSubcommands() []MDXSubcommand {
	var subcmds []MDXSubcommand

	for _, cmd := range c.Commands {
		subcmds = append(subcmds, MDXSubcommand{
			Name:        cmd.Name,
			Usage:       cmd.Usage,
			Description: c.extractFirstSentence(cmd.Description),
			Aliases:     cmd.Aliases,
		})
	}

	return subcmds
}

// extractFirstSentence gets the first sentence of a description
func (c *Command) extractFirstSentence(desc string) string {
	sentences := regexp.MustCompile(`[.!?]\s+`).Split(desc, 2)
	if len(sentences) > 0 {
		first := strings.TrimSpace(sentences[0])
		if first != "" && !strings.HasSuffix(first, ".") {
			first += "."
		}
		return first
	}

	lines := strings.Split(desc, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}

	return desc
}

// extractAllExamples parses examples from description
func (c *Command) extractAllExamples() []MDXExample {
	desc := c.Description
	var examples []MDXExample

	// Find EXAMPLES section
	exampleRegex := regexp.MustCompile(`(?s)EXAMPLES:\s*(.*?)(?:\n\n|\z)`)
	matches := exampleRegex.FindStringSubmatch(desc)
	if len(matches) < 2 {
		return examples
	}

	// Parse example lines
	lines := strings.Split(matches[1], "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Look for command lines containing the command name
		if strings.Contains(line, c.Name) {
			example := MDXExample{
				Command: c.cleanCommandLine(line),
				Comment: c.extractCommentFromLine(line),
				Title:   c.generateExampleTitle(line),
			}
			examples = append(examples, example)
		}
	}

	return examples
}

// extractAllFlags processes all flags into MDX format
func (c *Command) extractAllFlags() []MDXFlag {
	var flags []MDXFlag

	for _, flag := range c.Flags {
		mdxFlag := MDXFlag{
			Name:        flag.Name(),
			Description: flag.Usage(),
			Type:        c.getFlagTypeName(flag),
			Default:     c.getFlagDefaultValue(flag),
			EnvVar:      c.getFlagEnvVar(flag),
			Required:    flag.Required(),
			Category:    c.categorizeFlag(flag.Name()),
			Icon:        c.getFlagIcon(flag.Name()),
		}
		flags = append(flags, mdxFlag)
	}

	return flags
}

// categorizeFlag determines flag category based on name patterns
func (c *Command) categorizeFlag(name string) string {
	// Core configuration flags
	corePatterns := []string{"config", "workspace", "project", "database", "dsn", "url", "init", "force"}
	for _, pattern := range corePatterns {
		if strings.Contains(name, pattern) {
			return "core"
		}
	}

	// Build/deployment flags
	buildPatterns := []string{"build", "docker", "context", "registry", "image", "dockerfile"}
	for _, pattern := range buildPatterns {
		if strings.Contains(name, pattern) {
			return "build"
		}
	}

	// Development/debug flags
	devPatterns := []string{"verbose", "debug", "skip", "dry", "test", "branch", "commit"}
	for _, pattern := range devPatterns {
		if strings.Contains(name, pattern) {
			return "development"
		}
	}

	// Service/infrastructure flags
	servicePatterns := []string{"slack", "webhook", "notification", "auth", "token", "control"}
	for _, pattern := range servicePatterns {
		if strings.Contains(name, pattern) {
			return "service"
		}
	}

	return "core"
}

// determineCommandType categorizes the command
func (c *Command) determineCommandType() string {
	if len(c.Commands) > 0 {
		return "parent"
	}
	if c.Action != nil {
		return "leaf"
	}
	return "service"
}

// getFlagIcon assigns appropriate icons
func (c *Command) getFlagIcon(name string) string {
	iconMap := map[string]string{
		"config":    "folder",
		"workspace": "building",
		"project":   "box",
		"database":  "database",
		"url":       "link",
		"dsn":       "database",
		"webhook":   "webhook",
		"slack":     "message-circle",
		"verbose":   "terminal",
		"debug":     "bug",
		"docker":    "package",
		"build":     "hammer",
		"context":   "folder-open",
		"registry":  "archive",
		"auth":      "key",
		"token":     "shield",
		"init":      "plus",
		"force":     "alert-triangle",
	}

	for key, icon := range iconMap {
		if strings.Contains(name, key) {
			return icon
		}
	}

	return "settings"
}

// Helper methods
func (c *Command) cleanCommandLine(line string) string {
	if idx := strings.Index(line, "#"); idx != -1 {
		line = line[:idx]
	}
	line = strings.TrimSpace(line)
	line = regexp.MustCompile(`\s*\\\s*`).ReplaceAllString(line, " \\\n                   ")
	return line
}

func (c *Command) extractCommentFromLine(line string) string {
	if idx := strings.Index(line, "#"); idx != -1 {
		return strings.TrimSpace(line[idx+1:])
	}
	return ""
}

func (c *Command) generateExampleTitle(command string) string {
	if strings.Contains(command, "--help") {
		return "Show help"
	}
	if strings.Contains(command, "--init") {
		if strings.Contains(command, "--force") {
			return "Force initialize"
		}
		return "Initialize configuration"
	}
	if strings.Contains(command, "--docker-image") {
		return "Deploy pre-built image"
	}
	if strings.Contains(command, "--skip-push") {
		return "Local testing"
	}
	if strings.Contains(command, "--config=") {
		return "Custom config location"
	}
	if strings.Contains(command, "--registry=") {
		return "Custom registry"
	}
	if strings.Contains(command, "--context=") {
		return "Custom build context"
	}

	flagCount := strings.Count(command, "--")
	if flagCount > 3 {
		return "Advanced configuration"
	} else if flagCount > 1 {
		return "With options"
	}

	parts := strings.Fields(command)
	if len(parts) > 2 {
		return fmt.Sprintf("Run %s", parts[2])
	}

	return "Basic usage"
}

func (c *Command) getFlagTypeName(flag Flag) string {
	switch flag.(type) {
	case *StringFlag:
		return "string"
	case *BoolFlag:
		return "boolean"
	case *IntFlag:
		return "integer"
	case *FloatFlag:
		return "float"
	case *StringSliceFlag:
		return "string[]"
	default:
		return "unknown"
	}
}

func (c *Command) getFlagDefaultValue(flag Flag) string {
	switch f := flag.(type) {
	case *StringFlag:
		if val := f.Value(); val != "" {
			return val
		}
	case *BoolFlag:
		if f.HasValue() && f.Value() {
			return "true"
		}
	case *IntFlag:
		if f.HasValue() && f.Value() != 0 {
			return fmt.Sprintf("%d", f.Value())
		}
	case *FloatFlag:
		if f.HasValue() && f.Value() != 0.0 {
			return fmt.Sprintf("%.2f", f.Value())
		}
	case *StringSliceFlag:
		if val := f.Value(); len(val) > 0 {
			return strings.Join(val, ", ")
		}
	}
	return ""
}

func (c *Command) getFlagEnvVar(flag Flag) string {
	switch f := flag.(type) {
	case *StringFlag:
		return f.EnvVar()
	case *BoolFlag:
		return f.EnvVar()
	case *IntFlag:
		return f.EnvVar()
	case *FloatFlag:
		return f.EnvVar()
	case *StringSliceFlag:
		return f.EnvVar()
	}
	return ""
}

func (c *Command) extractEnvVariables() []MDXEnvVar {
	var envVars []MDXEnvVar
	seen := make(map[string]bool)

	for _, flag := range c.Flags {
		if envVar := c.getFlagEnvVar(flag); envVar != "" && !seen[envVar] {
			envVars = append(envVars, MDXEnvVar{
				Name:        envVar,
				Description: fmt.Sprintf("Default value for --%s flag", flag.Name()),
				Type:        c.getFlagTypeName(flag),
			})
			seen[envVar] = true
		}
	}

	return envVars
}

func (c *Command) hasItems(items any) bool {
	switch v := items.(type) {
	case []MDXFlag:
		return len(v) > 0
	case []MDXExample:
		return len(v) > 0
	case []MDXSubcommand:
		return len(v) > 0
	case []MDXEnvVar:
		return len(v) > 0
	default:
		return false
	}
}

// Clean, simple MDX template with tables for flags and frontmatter
const mdxTemplate = `---
{{- if .FrontMatter }}
title: "{{ .FrontMatter.Title }}"
description: "{{ .FrontMatter.Description }}"
{{- else }}
title: "{{ .Name | title }} Command"
description: "{{ .Usage }}"
{{- end }}
---

# {{ if .FrontMatter }}{{ .FrontMatter.Title }}{{ else }}{{ .Name | title }}{{ end }}

{{ .Usage }}

{{- if .HasSubcmds }}

> **Info:** This command has subcommands. Use ` + "`{{ .Name }} <subcommand>`" + ` to run specific services.

## Available Subcommands

{{- range .Subcommands }}
### {{ .Name }}

{{ .Usage }}
{{- if .Aliases }}

**Aliases:** ` + "`{{ join .Aliases \", \" }}`" + `
{{- end }}

{{- end }}

{{- else }}

## Usage

{{ .Description }}

{{- end }}

## Command Syntax

` + "```bash" + `
{{ .Name }} [flags]
` + "```" + `

{{- if hasItems .Examples }}

## Examples

{{- range .Examples }}

### {{ .Title }}

` + "```bash" + `
{{ .Command }}
` + "```" + `
{{- end }}

{{- end }}

{{- if hasItems .Flags }}

## Flags

### Configuration

| Flag | Description | Type | Default | Environment |
| --- | --- | --- | --- | --- |
{{- range .Flags }}
{{- if eq .Category "core" }}
| ` + "`--{{ .Name }}`" + `{{- if .Required }} **(required)**{{ end }} | {{ .Description }} | {{ .Type }} | {{- if .Default }}` + "`{{ .Default }}`" + `{{- else }}-{{- end }} | {{- if .EnvVar }}` + "`{{ .EnvVar }}`" + `{{- else }}-{{- end }} |
{{- end }}
{{- end }}

{{- $hasBuild := false }}
{{- range .Flags }}{{- if eq .Category "build" }}{{- $hasBuild = true }}{{- end }}{{- end }}
{{- if $hasBuild }}

### Build & Deployment

| Flag | Description | Type | Default | Environment |
| --- | --- | --- | --- | --- |
{{- range .Flags }}
{{- if eq .Category "build" }}
| ` + "`--{{ .Name }}`" + ` | {{ .Description }} | {{ .Type }} | {{- if .Default }}` + "`{{ .Default }}`" + `{{- else }}-{{- end }} | {{- if .EnvVar }}` + "`{{ .EnvVar }}`" + `{{- else }}-{{- end }} |
{{- end }}
{{- end }}

{{- end }}

{{- $hasDev := false }}
{{- range .Flags }}{{- if eq .Category "development" }}{{- $hasDev = true }}{{- end }}{{- end }}
{{- if $hasDev }}

### Development Options

| Flag | Description | Type | Default |
| --- | --- | --- | --- |
{{- range .Flags }}
{{- if eq .Category "development" }}
| ` + "`--{{ .Name }}`" + ` | {{ .Description }} | {{ .Type }} | {{- if .Default }}` + "`{{ .Default }}`" + `{{- else }}-{{- end }} |
{{- end }}
{{- end }}

{{- end }}

{{- $hasService := false }}
{{- range .Flags }}{{- if eq .Category "service" }}{{- $hasService = true }}{{- end }}{{- end }}
{{- if $hasService }}

### Service Configuration

| Flag | Description | Type | Default | Environment |
| --- | --- | --- | --- | --- |
{{- range .Flags }}
{{- if eq .Category "service" }}
| ` + "`--{{ .Name }}`" + ` | {{ .Description }} | {{ .Type }} | {{- if .Default }}` + "`{{ .Default }}`" + `{{- else }}-{{- end }} | {{- if .EnvVar }}` + "`{{ .EnvVar }}`" + `{{- else }}-{{- end }} |
{{- end }}
{{- end }}

{{- end }}

{{- end }}

{{- if hasItems .EnvVars }}

## Environment Variables

| Variable | Description | Type |
| --- | --- | --- |
{{- range .EnvVars }}
| ` + "`{{ .Name }}`" + ` | {{ .Description }} | {{ .Type }} |
{{- end }}

{{- end }}`
