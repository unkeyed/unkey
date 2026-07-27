package validation

import (
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

// Walking the JSON Schema failure tree: which nodes are reported, and which are
// suppressed because a node above them already said it better.

// fromSchemaFailures walks the flattened JSON Schema output, which pairs each
// failure's typed kind with its instance and schema pointers.
//
// The flattening lists a node before its causes, which is what lets a branching
// combinator suppress them: 'anyOf' and 'oneOf' are followed by the failure of
// every alternative they tried, and those contradict each other, so a caller told
// that a value must be a string and must be an integer learns less than one told
// it matched no allowed alternative.
func fromSchemaFailures(prefix string, root *jsonschema.ValidationError, allowed *propertyResolver) []ValidationError {
	output := root.BasicOutput()

	units := output.Errors
	if len(units) == 0 && output.Error != nil {
		units = []jsonschema.OutputUnit{*output}
	}

	errs := make([]ValidationError, 0, len(units))
	var pruned []string

	// A choice that only expresses nullability is not a choice, so its real
	// failures are reported instead of the choice itself.
	nullable := nullableChoices(units)

	for _, unit := range units {
		if unit.Error == nil {
			continue
		}
		if isPruned(pruned, unit.KeywordLocation) {
			continue
		}

		if surviving, ok := nullable[unit.KeywordLocation]; ok {
			for _, branch := range branchPrefixes(units, unit.KeywordLocation) {
				if branch != surviving {
					pruned = append(pruned, branch)
				}
			}

			continue
		}

		described := describeKind(unit.Error.Kind, func() []string {
			return allowed.lookup(unit.KeywordLocation)
		})
		if described.skip {
			continue
		}
		if described.prune {
			pruned = append(pruned, unit.KeywordLocation)
		}

		location := renderLocation(prefix, unit.InstanceLocation, unit.KeywordLocation)

		if len(described.expand) == 0 {
			errs = append(errs, newError(location, described.message, described.fix))

			continue
		}

		// 'required' reports the whole object; a caller wants to be pointed at each
		// property that is missing.
		for _, name := range described.expand {
			property, ok := specToken(name)
			if !ok {
				continue
			}
			errs = append(errs, newError(location+"."+property, described.message, described.fix))
		}
	}

	return errs
}

// nullableChoices finds the 'anyOf' and 'oneOf' nodes that exist only to make a
// value nullable, and returns the one branch worth reporting for each.
//
// `anyOf: [{$ref: Healthcheck}, {type: "null"}]` is how an optional object is
// spelled in OpenAPI 3.1, and it is a choice only in a technical sense: a caller
// who sent a malformed Healthcheck wants to hear which field is wrong, not that
// their value "must match at least one of the schemas defined for it". The null
// branch fails with a type error wanting exactly null, so a choice where all but
// one branch failed that way carries no real alternatives, and suppressing its
// children would be a pure loss of detail.
//
// A choice with two or more surviving branches keeps the summary, because there
// the branches genuinely contradict each other.
func nullableChoices(units []jsonschema.OutputUnit) map[string]string {
	var choices map[string]string

	for _, unit := range units {
		if unit.Error == nil {
			continue
		}
		switch unit.Error.Kind.(type) {
		case *kind.AnyOf, *kind.OneOf:
		default:
			continue
		}

		surviving := ""
		count := 0
		for _, branch := range branchPrefixes(units, unit.KeywordLocation) {
			if nullOnlyBranch(units, branch) {
				continue
			}
			surviving = branch
			count++
		}

		if count != 1 {
			continue
		}
		if choices == nil {
			choices = make(map[string]string)
		}
		choices[unit.KeywordLocation] = surviving
	}

	return choices
}

// branchPrefixes returns the keyword locations of a choice's alternatives, in
// order, deduplicated: for '/properties/x/anyOf' those are '/properties/x/anyOf/0'
// and '/properties/x/anyOf/1'.
func branchPrefixes(units []jsonschema.OutputUnit, choice string) []string {
	seen := make(map[string]bool)
	var prefixes []string

	for _, unit := range units {
		if !strings.HasPrefix(unit.KeywordLocation, choice+"/") {
			continue
		}

		rest := unit.KeywordLocation[len(choice)+1:]
		index := rest
		if cut := strings.IndexByte(rest, '/'); cut >= 0 {
			index = rest[:cut]
		}
		if !isIndex(index) {
			continue
		}

		branch := choice + "/" + index
		if seen[branch] {
			continue
		}
		seen[branch] = true
		prefixes = append(prefixes, branch)
	}

	return prefixes
}

// nullOnlyBranch reports whether every failure under branch is the value not
// being null, which is what the null alternative of a nullable schema produces.
func nullOnlyBranch(units []jsonschema.OutputUnit, branch string) bool {
	failures := 0

	for _, unit := range units {
		if unit.Error == nil {
			continue
		}
		if unit.KeywordLocation != branch && !strings.HasPrefix(unit.KeywordLocation, branch+"/") {
			continue
		}

		switch k := unit.Error.Kind.(type) {
		case *kind.Group, *kind.Schema, *kind.Reference:
			// Structural, carries no constraint of its own.
		case *kind.Type:
			if len(k.Want) != 1 || k.Want[0] != "null" {
				return false
			}
			failures++
		default:
			return false
		}
	}

	return failures > 0
}

// isIndex reports whether s is a decimal branch index.
func isIndex(s string) bool {
	if s == "" {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}

	return true
}

// isPruned reports whether a keyword location sits underneath one that has
// already been reported as a whole.
func isPruned(pruned []string, keywordLocation string) bool {
	for _, prefix := range pruned {
		if strings.HasPrefix(keywordLocation, prefix+"/") {
			return true
		}
	}

	return false
}
