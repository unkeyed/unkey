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
	lines := strings.SplitSeq(matches[1], "\n")
	for line := range lines {
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

// determineCommandType categorizes the command for template selection
// Examples:
//   - "parent": `unkey deploy` (has subcommands like `unkey deploy init`)
//   - "leaf": `unkey version` (executes an action, no subcommands)
//   - "service": configuration-only commands with no action or subcommands
func (c *Command) determineCommandType() string {
	if len(c.Commands) > 0 {
		return "parent"
	}
	if c.Action != nil {
		return "leaf"
	}
	return "service"
}

// cleanCommandLine formats command examples for documentation display
// Removes inline comments and formats line continuations with proper indentation
// Example input:  "unkey deploy \     --workspace-id=ws_123 \     --verbose  # deploy with options"
//
//	Example output: "unkey deploy \
//	                   --workspace-id=ws_123 \
//	                   --verbose"
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

// extractCommentFromLine pulls out inline comments from command examples
// Used to extract explanatory text that appears after # in example commands
// Example: "unkey deploy --verbose  # This enables detailed logging" → "This enables detailed logging"
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

// extractExampleTitlesFromDescription parses example titles from various markdown patterns
// Looks for titles that precede code blocks in the command description to create a mapping
// of commands to their explicit titles, avoiding generic fallback titles when possible
// Example patterns it matches:
//   - "#### Initialize new project\n```bash\nunkey deploy --init\n```"
//   - "**Deploy with options:**\n```bash\nunkey deploy --verbose\n```"
//   - "Standard deployment:\n```bash\nunkey deploy\n```"
func (c *Command) extractExampleTitlesFromDescription() map[string]string {
	if c.Description == "" {
		return nil
	}

	titleMap := make(map[string]string)

	// Try multiple patterns to find title-command pairs
	backtick := "`"
	patterns := []string{
		// #### Title followed by ```bash
		`(?s)#{3,4}\s*([^\n]+)\n\s*` + backtick + `{3}bash\s*\n([^` + backtick + `]+)` + backtick + `{3}`,
		// **Title:** followed by ```bash
		`(?s)\*\*([^*]+):\*\*\s*\n\s*` + backtick + `{3}bash\s*\n([^` + backtick + `]+)` + backtick + `{3}`,
		// Title: followed by ```bash
		`(?s)([^:\n]+):\s*\n\s*` + backtick + `{3}bash\s*\n([^` + backtick + `]+)` + backtick + `{3}`,
		// Any heading (# ## ### ####) followed by bash block
		`(?s)#{1,4}\s*([^\n]+)\n[^` + backtick + `]*` + backtick + `{3}bash\s*\n([^` + backtick + `]+)` + backtick + `{3}`,
	}

	for _, pattern := range patterns {
		regex := regexp.MustCompile(pattern)
		matches := regex.FindAllStringSubmatch(c.Description, -1)

		for _, match := range matches {
			if len(match) >= 3 {
				title := strings.TrimSpace(match[1])
				command := strings.TrimSpace(match[2])
				cleanCommand := c.cleanCommandLine(command)
				titleMap[cleanCommand] = title
			}
		}
	}

	return titleMap
}

// generateExampleTitle generates titles using extracted examples or fallback logic
// Prioritizes explicit titles from the command description over generic fallbacks
// This ensures documentation shows meaningful titles like "Initialize new project"
// instead of generic ones like "With options"
func (c *Command) generateExampleTitle(command string) string {
	if command == "" {
		return "Basic usage"
	}

	// First try to get title from parsed examples
	titleMap := c.extractExampleTitlesFromDescription()
	cleanCommand := c.cleanCommandLine(command)

	if title, exists := titleMap[cleanCommand]; exists {
		return title
	}

	// Fallback here only if not found in description
	return c.generateFallbackTitle(command)
}

// generateFallbackTitle provides simple fallback logic when no explicit title is found
// Creates generic but useful titles based on command complexity and structure
// Examples:
//   - "unkey deploy --init --force --verbose" → "Advanced configuration" (3+ flags)
//   - "unkey deploy --verbose" → "With options" (1-2 flags)
//   - "unkey version check" → "Run check" (has subcommand)
//   - "unkey deploy" → "Basic usage" (no flags)
func (c *Command) generateFallbackTitle(command string) string {
	if command == "" {
		return "Basic usage"
	}

	// Simple fallback based on complexity
	flagCount := strings.Count(command, "--")
	if flagCount > 3 {
		return "Advanced configuration"
	}
	if flagCount > 1 {
		return "With options"
	}

	// Try to extract subcommand name
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
