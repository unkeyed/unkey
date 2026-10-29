package discovery

import (
	"encoding/json"
	"fmt"
	"slices"
	"strings"

	"github.com/unkeyed/unkey/pkg/cli"
)

// Scope excludes orchestration commands and the legacy API aliases.
const Scope = "canonical API resource commands (excludes login and deploy orchestration)"

// Entry identifies one canonical leaf command.
type Entry struct {
	Path     string   `json:"path"`
	Usage    string   `json:"usage"`
	Endpoint Endpoint `json:"endpoint"`
}

// Catalog lists canonical API commands in lexical path order.
type Catalog struct {
	Version  int     `json:"version"`
	Scope    string  `json:"scope"`
	Commands []Entry `json:"commands"`
}

// Contract combines declared CLI metadata with the authoritative request schema.
type Contract struct {
	Version int    `json:"version"`
	Scope   string `json:"scope"`
	Entry
	Description  string   `json:"description"`
	Examples     []string `json:"examples"`
	Flags        []Flag   `json:"flags"`
	RequireOneOf []Choice `json:"requireOneOf,omitempty"`
	Defaults     string   `json:"flagDefaults"`
	Request
}

// Choice requires exactly one flag unless an explicit alternative waives it.
type Choice struct {
	Flags  []string `json:"flags"`
	Unless []string `json:"unless,omitempty"`
}

// Flag describes declared input, never its current or environment-derived value.
type Flag struct {
	Name           string          `json:"name"`
	Type           string          `json:"type"`
	Usage          string          `json:"usage"`
	Required       bool            `json:"required"`
	RequiredUnless []string        `json:"requiredUnless,omitempty"`
	Conflicts      []string        `json:"conflicts,omitempty"`
	Environment    string          `json:"environment,omitempty"`
	Enum           []string        `json:"enum,omitempty"`
	Encoding       string          `json:"encoding,omitempty"`
	RequestFields  []string        `json:"requestFields,omitempty"`
	Schema         json.RawMessage `json:"schema,omitempty"`
	AtMostOneField []string        `json:"atMostOneField,omitempty"`
}

func List(commands []*cli.Command, paths map[string]string) (Catalog, error) {
	catalog := Catalog{Version: 1, Scope: Scope, Commands: make([]Entry, 0, len(paths))}
	for legacy, path := range paths {
		command := find(commands, strings.Fields(path))
		if command == nil || command.Action == nil {
			return Catalog{}, fmt.Errorf("canonical API command %q not found", path)
		}
		endpoint, _, err := lookup(legacy)
		if err != nil {
			return Catalog{}, err
		}
		catalog.Commands = append(catalog.Commands, Entry{Path: path, Usage: command.Usage, Endpoint: endpoint})
	}
	slices.SortFunc(catalog.Commands, func(a, b Entry) int { return strings.Compare(a.Path, b.Path) })
	return catalog, nil
}

// Describe returns one leaf contract; group paths and noncanonical aliases fail.
func Describe(commands []*cli.Command, paths map[string]string, path string) (Contract, error) {
	legacy := ""
	for key, canonical := range paths {
		if canonical == path {
			legacy = key
			break
		}
	}
	command := find(commands, strings.Fields(path))
	if legacy == "" || command == nil || command.Action == nil {
		return Contract{}, fmt.Errorf("unknown canonical API command %q; run 'unkey describe' to list commands", path)
	}
	endpoint, request, err := lookup(legacy)
	if err != nil {
		return Contract{}, err
	}
	flags, err := describeFlags(command, request, legacy)
	if err != nil {
		return Contract{}, fmt.Errorf("describe %s: %w", path, err)
	}
	return Contract{
		Version: 1, Scope: Scope, Entry: Entry{Path: path, Usage: command.Usage, Endpoint: endpoint},
		Description: command.Description, Examples: command.Examples, Flags: flags,
		RequireOneOf: choices(command),
		Defaults:     "omitted: the CLI does not expose declared defaults separately from environment-derived values; schema defaults are API annotations",
		Request:      request,
	}, nil
}

func choices(command *cli.Command) []Choice {
	var result []Choice
	for _, group := range command.RequireOneOf {
		choice := Choice{Flags: group, Unless: nil}
		for _, flag := range command.Flags {
			if slices.Contains(group, flag.Name()) {
				continue
			}
			conflicts := command.ConflictingFlags(flag.Name())
			if !slices.ContainsFunc(group, func(name string) bool { return !slices.Contains(conflicts, name) }) {
				choice.Unless = append(choice.Unless, flag.Name())
			}
		}
		result = append(result, choice)
	}
	return result
}

func find(commands []*cli.Command, parts []string) *cli.Command {
	for _, command := range commands {
		if len(parts) == 0 || command.Name != parts[0] {
			continue
		}
		if len(parts) == 1 {
			return command
		}
		return find(command.Commands, parts[1:])
	}
	return nil
}
