package cli

import (
	"errors"
	"fmt"
	"regexp"
	"strings"
	"text/template"

	"golang.org/x/text/cases"
	"golang.org/x/text/language"
)

var (
	ErrCommandNil           = errors.New("command cannot be nil")
	ErrTemplateParseFailure = errors.New("failed to parse MDX template")
	ErrTemplateExecFailure  = errors.New("failed to execute MDX template")
)

// FrontMatter holds metadata for the MDX file
type FrontMatter struct {
	Title       string
	Description string
}

// GenerateMDX creates Fuma docs MDX from any command metadata with frontmatter
func (c *Command) GenerateMDX(frontMatter *FrontMatter) (string, error) {
	if c == nil {
		return "", ErrCommandNil
	}

	data := c.extractMDXData(frontMatter)

	// Choose template based on whether this is a parent command or leaf command
	var templateStr string
	if len(c.Commands) > 0 {
		templateStr = parentCommandTemplate
	} else {
		templateStr = leafCommandTemplate
	}

	caser := cases.Title(language.English)
	tmpl, err := template.New("mdx").Funcs(template.FuncMap{
		"join":      strings.Join,
		"contains":  strings.Contains,
		"hasPrefix": strings.HasPrefix,
		"title":     caser.String,
		"lower":     strings.ToLower,
		"hasItems":  func(items any) bool { return c.hasItems(items) },
		"eq":        func(a, b string) bool { return a == b },
	}).Parse(templateStr)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTemplateParseFailure, err)
	}

	var result strings.Builder
	err = tmpl.Execute(&result, data)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrTemplateExecFailure, err)
	}

	return result.String(), nil
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
	if desc == "" {
		return ""
	}

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
	if c.Description == "" {
		return nil
	}

	// Find EXAMPLES section
	exampleRegex := regexp.MustCompile(`(?s)EXAMPLES:\s*(.*?)(?:\n\n|\z)`)
	matches := exampleRegex.FindStringSubmatch(c.Description)
	if len(matches) < 2 {
		return nil
	}

	var examples []MDXExample

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
	if len(c.Flags) == 0 {
		return nil
	}

	var flags []MDXFlag

	for _, flag := range c.Flags {
		if flag == nil {
			continue
		}

		mdxFlag := MDXFlag{
			Name:        flag.Name(),
			Description: flag.Usage(),
			Type:        c.getTypeString(flag),
			Default:     c.getDefaultValue(flag),
			EnvVar:      c.getEnvVar(flag),
			Required:    flag.Required(),
		}
		flags = append(flags, mdxFlag)
	}

	return flags
}

// getTypeString returns the type name for a flag
func (c *Command) getTypeString(flag Flag) string {
	if flag == nil {
		return "unknown"
	}

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

// Helper methods
func (c *Command) cleanCommandLine(line string) string {
	if line == "" {
		return ""
	}

	if idx := strings.Index(line, "#"); idx != -1 {
		line = line[:idx]
	}
	line = strings.TrimSpace(line)
	line = regexp.MustCompile(`\s*\\\s*`).ReplaceAllString(line, " \\\n                   ")
	return line
}

func (c *Command) extractCommentFromLine(line string) string {
	if line == "" {
		return ""
	}

	idx := strings.Index(line, "#")
	if idx == -1 {
		return ""
	}

	return strings.TrimSpace(line[idx+1:])
}

func (c *Command) generateExampleTitle(command string) string {
	if command == "" {
		return "Basic usage"
	}

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

func (c *Command) extractEnvVariables() []MDXEnvVar {
	if len(c.Flags) == 0 {
		return nil
	}

	var envVars []MDXEnvVar
	seen := make(map[string]bool)

	for _, flag := range c.Flags {
		envVar := c.getEnvVar(flag)
		if envVar != "" && !seen[envVar] {
			envVars = append(envVars, MDXEnvVar{
				Name:        envVar,
				Description: fmt.Sprintf("Default value for --%s flag", flag.Name()),
				Type:        c.getTypeString(flag),
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

// Template for parent commands (commands with subcommands)
const parentCommandTemplate = `---
{{- if .FrontMatter }}
title: "{{ .FrontMatter.Title }}"
description: "{{ .FrontMatter.Description }}"
{{- else }}
title: "{{ .Name | title }} Command"
description: "{{ .Usage }}"
{{- end }}
---
{{- if .Description }}
{{ .Description }}
{{- else }}
{{ .Usage }}
{{- end }}

## Quick Reference

{{- range .Subcommands }}
- ` + "`{{ $.Name }} {{ .Name }}`" + ` - {{ .Usage }}
{{- end }}

{{- if hasItems .Flags }}

## Global Flags

These flags apply to all subcommands:

| Flag | Description | Type | Default | Environment |
| --- | --- | --- | --- | --- |
{{- range .Flags }}
| ` + "`--{{ .Name }}`" + `{{- if .Required }} **(required)**{{ end }} | {{ .Description }} | {{ .Type }} | {{- if .Default }}` + "`{{ .Default }}`" + `{{- else }}-{{- end }} | {{- if .EnvVar }}` + "`{{ .EnvVar }}`" + `{{- else }}-{{- end }} |
{{- end }}

{{- end }}`

// Template for leaf commands (commands without subcommands)
const leafCommandTemplate = `---
{{- if .FrontMatter }}
title: "{{ .FrontMatter.Title }}"
description: "{{ .FrontMatter.Description }}"
{{- else }}
title: "{{ .Name | title }} Command"
description: "{{ .Usage }}"
{{- end }}
---

{{- if .Description }}
{{ .Description }}
{{- else }}
{{ .Usage }}
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

| Flag | Description | Type | Default | Environment |
| --- | --- | --- | --- | --- |
{{- range .Flags }}
| ` + "`--{{ .Name }}`" + `{{- if .Required }} **(required)**{{ end }} | {{ .Description }} | {{ .Type }} | {{- if .Default }}` + "`{{ .Default }}`" + `{{- else }}-{{- end }} | {{- if .EnvVar }}` + "`{{ .EnvVar }}`" + `{{- else }}-{{- end }} |
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
