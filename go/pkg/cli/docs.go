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
	CommandType string
	FrontMatter *FrontMatter
	CommandPath string
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

// extractMDXData parses command metadata into structured data
func (c *Command) extractMDXData(frontMatter *FrontMatter) MDXData {
	return MDXData{
		Name:        c.Name,
		Usage:       c.Usage,
		Description: c.getCleanDescription(),
		HasSubcmds:  len(c.Commands) > 0,
		Subcommands: c.extractSubcommands(),
		Examples:    c.extractExamples(),
		Flags:       c.extractFlags(),
		CommandType: c.determineCommandType(),
		FrontMatter: frontMatter,
		CommandPath: c.getCommandPath(),
	}
}

// getCleanDescription returns description without EXAMPLES section for MDX
// and converts UPPERCASE sections to proper markdown headings
func (c *Command) getCleanDescription() string {
	if c.Description == "" {
		return c.Usage
	}

	// Remove EXAMPLES section for cleaner MDX description
	exampleRegex := regexp.MustCompile(`(?s)\nEXAMPLES:.*`)
	cleaned := exampleRegex.ReplaceAllString(c.Description, "")

	// Convert UPPERCASE SECTIONS: to ## headings
	headingRegex := regexp.MustCompile(`\n([A-Z][A-Z\s]+):\s*\n`)
	cleaned = headingRegex.ReplaceAllStringFunc(cleaned, func(match string) string {
		// Extract the heading text
		parts := headingRegex.FindStringSubmatch(match)
		if len(parts) > 1 {
			heading := strings.TrimSpace(parts[1])
			// Convert to title case for better readability
			caser := cases.Title(language.English)
			heading = caser.String(strings.ToLower(heading))
			return fmt.Sprintf("\n## %s\n\n", heading)
		}
		return match
	})

	return strings.TrimSpace(cleaned)
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

// extractFirstSentence gets the first sentence of a description for use in summary contexts
func (c *Command) extractFirstSentence(desc string) string {
	if desc == "" {
		return ""
	}

	// Split on sentence-ending punctuation followed by whitespace
	sentences := regexp.MustCompile(`[.!?]\s+`).Split(desc, 2)
	if len(sentences) > 0 {
		first := strings.TrimSpace(sentences[0])
		if first != "" && !strings.HasSuffix(first, ".") {
			first += "."
		}
		return first
	}

	// Fallback: use first line if no sentence punctuation found
	lines := strings.Split(desc, "\n")
	if len(lines) > 0 {
		return strings.TrimSpace(lines[0])
	}

	return desc
}

// extractAllExamples parses examples from a dedicated EXAMPLES section in the command description
// Looks for a section starting with "EXAMPLES:" and extracts command lines that contain the command name
// This allows embedding rich examples directly in the CLI source code that get formatted for documentation
// Example description format:
//
//	"Deploy applications to Unkey infrastructure.
//
//	 EXAMPLES:
//	 unkey deploy --init          # Initialize configuration
//	 unkey deploy --verbose       # Deploy with detailed output
//	 unkey deploy --skip-push     # Local testing only"
func (c *Command) extractExamples() []MDXExample {
	if c.Description == "" {
		return nil
	}

	// Find EXAMPLES section
	exampleRegex := regexp.MustCompile(`(?s)EXAMPLES:\s*(.*?)(?:\n[A-Z][A-Z\s]*:|\z)`)
	matches := exampleRegex.FindStringSubmatch(c.Description)
	if len(matches) < 2 {
		return nil
	}

	var examples []MDXExample
	// Parse example lines from the EXAMPLES section
	lines := strings.SplitSeq(matches[1], "\n")

	for line := range lines {
		line = strings.TrimSpace(line)

		// Skip empty lines and comment-only lines
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}

		// Only process lines that contain the command name
		if strings.Contains(line, c.Name) {
			example := MDXExample{
				Command: c.cleanCommand(line),
				Comment: c.extractComment(line),
				Title:   c.generateTitle(line),
			}
			examples = append(examples, example)
		}
	}

	return examples
}

// cleanCommand removes comments and cleans up command line
func (c *Command) cleanCommand(line string) string {
	if line == "" {
		return ""
	}

	// Remove inline comments
	if idx := strings.Index(line, "#"); idx != -1 {
		line = line[:idx]
	}

	return strings.TrimSpace(line)
}

// extractComment pulls out inline comments from command examples
func (c *Command) extractComment(line string) string {
	if line == "" {
		return ""
	}

	idx := strings.Index(line, "#")
	if idx == -1 {
		return ""
	}

	return strings.TrimSpace(line[idx+1:])
}

// generateTitle creates meaningful titles for examples
func (c *Command) generateTitle(command string) string {
	if command == "" {
		return "Basic usage"
	}

	// Always prioritize comment as title if present
	if comment := c.extractComment(command); comment != "" {
		// Capitalize first letter
		if len(comment) > 0 {
			return strings.ToUpper(comment[:1]) + comment[1:]
		}
	}

	// Simple fallback based on complexity
	cleanCmd := c.cleanCommand(command)
	flagCount := strings.Count(cleanCmd, "--")

	if flagCount == 0 {
		return "Basic usage"
	}
	if flagCount == 1 {
		return "Basic configuration"
	}
	if flagCount <= 3 {
		return "With options"
	}

	return "Advanced configuration"
}

// Get full command path
func (c *Command) getCommandPath() string {
	if c.commandPath != "" {
		return c.commandPath
	}
	return c.Name
}

// extractFlags to include short names and aliases
func (c *Command) extractFlags() []MDXFlag {
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
	case *DurationFlag:
		return "duration"
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
func (c *Command) determineCommandType() string {
	if len(c.Commands) > 0 {
		return "parent"
	}
	if c.Action != nil {
		return "leaf"
	}
	return "service"
}

// hasItems checks if any slice has items for template conditionals
func (c *Command) hasItems(items any) bool {
	switch v := items.(type) {
	case []MDXFlag:
		return len(v) > 0
	case []MDXExample:
		return len(v) > 0
	case []MDXSubcommand:
		return len(v) > 0
	default:
		return false
	}
}
