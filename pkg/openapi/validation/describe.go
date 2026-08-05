package validation

import (
	"math/big"
	"sort"
	"strconv"
	"strings"

	"github.com/santhosh-tekuri/jsonschema/v6"
	"github.com/santhosh-tekuri/jsonschema/v6/kind"
)

// describeKind turns one typed JSON Schema failure into a caller-facing message.
//
// Every kind in the jsonschema library carries the offending value alongside the
// constraint that rejected it: kind.Pattern has Got and Want, kind.MaxLength has
// the length that was sent and the limit, kind.AdditionalProperties has the
// property names the caller invented. This function reads only the constraint
// side and never the Got side, which is what keeps request data out of both the
// response and the request log. See the package comment in validator.go.
//
// The message is a predicate about the field named by ValidationError.Location,
// so it reads as a sentence when the two are joined: "body.variables[1].key must
// match the pattern '^[A-Za-z_][A-Za-z0-9_]*$'".
//
// allowed resolves the property names the specification declares next to the
// failing keyword, and is only consulted by the two keywords that reject a
// property the caller named: those are the cases where the message has to
// withhold what was wrong, so the fix has to say what would be right instead. It
// may be nil, and it is only called on the rejection path.
func describeKind(k jsonschema.ErrorKind, allowed propertyLookup) described {
	switch e := k.(type) {
	// Grouping nodes. They restate a failure their children already describe, or
	// they describe the schema rather than the request. 'allOf' and '$ref' merge
	// subschemas without branching, so the child failure is the whole story.
	case *kind.Schema, *kind.Group, *kind.AllOf, *kind.Reference, *kind.RefCycle:
		return structural()

	// The branching combinators are the opposite case: their children are the
	// failures of alternatives the caller was never required to satisfy all of.
	// Reporting them says the value must be a string and must be an integer, so
	// the choice is reported and the branches below it are dropped.
	case *kind.AnyOf:
		return choice("must match at least one of the schemas defined for it")

	case *kind.OneOf:
		// With no matching subschema the causes are the branch failures. With more
		// than one match there are no causes at all.
		if len(e.Subschemas) == 0 {
			return choice("must match exactly one of the schemas defined for it, but it matched none")
		}

		return choice("must match exactly one of the schemas defined for it, but it matched more than one")

	case *kind.Not:
		return choice("must not match the schema defined under 'not'")

	case *kind.FalseSchema:
		return msg("is not accepted here by the specification")

	case *kind.Type:
		want := joinWithOr(articled(e.Want))
		if want == "" {
			return msg("is not of a type this field accepts")
		}
		// Got is a JSON type name computed from the decoded value, never the value
		// itself, but it is only echoed when it is one of the names the library
		// can produce.
		if _, ok := jsonTypeNames[e.Got]; ok {
			return msg("must be " + want + ", but " + article(e.Got) + " was sent")
		}

		return msg("must be " + want)

	case *kind.Enum:
		values, ok := displaySpecValues(e.Want)
		if !ok {
			return msg("must be one of the values listed in the schema")
		}

		return msg("must be one of: " + values)

	case *kind.Const:
		value, ok := displaySpecValue(e.Want)
		if !ok {
			return msg("must equal the constant value defined in the schema")
		}

		return msg("must be " + value)

	case *kind.Format:
		if name, ok := specToken(e.Want); ok {
			return msg("must be a valid " + name + " value")
		}

		return msg("must match the format defined in the schema")

	case *kind.Required:
		// Missing is the schema's required list minus the properties that were
		// present, so every name here comes from the specification.
		return expand("is required", e.Missing)

	case *kind.Dependency:
		return dependencyDescription(e.Prop, e.Missing)

	case *kind.DependentRequired:
		return dependencyDescription(e.Prop, e.Missing)

	case *kind.AdditionalProperties:
		return undeclaredDescription(e.Properties, allowed)

	case *kind.PropertyNames:
		return msgFix("contains a property whose name the schema does not allow",
			renameUndeclaredFix(allowed))

	case *kind.MinProperties:
		return msg("must have at least " + plural(e.Want, "property", "properties"))

	case *kind.MaxProperties:
		return msg("must have at most " + plural(e.Want, "property", "properties"))

	case *kind.MinItems:
		return msg("must contain at least " + plural(e.Want, "item", "items"))

	case *kind.MaxItems:
		return msg("must contain at most " + plural(e.Want, "item", "items"))

	case *kind.AdditionalItems:
		return msg("must not contain more items than the schema defines")

	case *kind.UniqueItems:
		return msg("must not contain duplicate items, but the items at index " +
			strconv.Itoa(e.Duplicates[0]) + " and " + strconv.Itoa(e.Duplicates[1]) + " are equal")

	case *kind.Contains:
		return msg("must contain at least one item matching the schema")

	case *kind.MinContains:
		return msg("must contain at least " + plural(e.Want, "item", "items") + " matching the schema")

	case *kind.MaxContains:
		return msg("must contain at most " + plural(e.Want, "item", "items") + " matching the schema")

	case *kind.MinLength:
		return msg("must be at least " + plural(e.Want, "character", "characters") + " long")

	case *kind.MaxLength:
		return msg("must be at most " + plural(e.Want, "character", "characters") + " long")

	case *kind.Pattern:
		if pattern, ok := specToken(e.Want); ok {
			return msg("must match the pattern '" + pattern + "'")
		}

		return msg("must match the pattern defined in the schema")

	case *kind.ContentEncoding:
		if name, ok := specToken(e.Want); ok {
			return msg("must be encoded as " + name)
		}

		return msg("must use the encoding defined in the schema")

	case *kind.ContentMediaType:
		if name, ok := specToken(e.Want); ok {
			return msg("must decode to media type " + name)
		}

		return msg("must decode to the media type defined in the schema")

	case *kind.ContentSchema:
		return msg("must decode to a value matching the schema")

	case *kind.Minimum:
		return numericDescription("must be at least ", e.Want)

	case *kind.Maximum:
		return numericDescription("must be at most ", e.Want)

	case *kind.ExclusiveMinimum:
		return numericDescription("must be greater than ", e.Want)

	case *kind.ExclusiveMaximum:
		return numericDescription("must be less than ", e.Want)

	case *kind.MultipleOf:
		return numericDescription("must be a multiple of ", e.Want)

	case *kind.InvalidJsonValue:
		return msg("is not a value that can be validated as JSON")

	default:
		// A library upgrade that adds a keyword lands here. Losing detail is the
		// intended outcome; guessing at the wording of a message that may embed
		// the value is not.
		return msg(fallbackMessage)
	}
}

// msg reports a failure with nothing to add beyond the predicate.
func msg(message string) described {
	return described{message: message, fix: "", expand: nil, prune: false, skip: false}
}

// msgFix reports a failure along with a remediation hint, which is how the cases
// that had to withhold a name from the request stay actionable.
func msgFix(message, fix string) described {
	return described{message: message, fix: fix, expand: nil, prune: false, skip: false}
}

// expand reports one failure per name, at "<location>.<name>".
func expand(message string, names []string) described {
	return described{message: message, fix: "", expand: names, prune: false, skip: false}
}

// choice reports a branching combinator, and suppresses the failures of the
// branches it tried.
func choice(message string) described {
	return described{message: message, fix: "", expand: nil, prune: true, skip: false}
}

// structural reports a node that describes the schema rather than the request.
func structural() described {
	return described{message: "", fix: "", expand: nil, prune: false, skip: true}
}

// described is one rendered failure, before it is attached to a location.
type described struct {
	// message is the predicate to report, or the empty string when skip is set.
	message string

	// fix is an optional remediation hint, set only where the message had to
	// withhold specifics that came from the request.
	fix string

	// expand names the properties to fan this failure out over, producing one
	// error per name at "<location>.<name>". Only 'required' uses it.
	expand []string

	// prune drops every failure reported underneath this one. Set for the
	// combinators whose children are alternatives rather than requirements.
	prune bool

	// skip marks a node that describes the schema's structure rather than a
	// problem the caller can act on.
	skip bool
}

// fallbackMessage is what a caller gets when the library reports a failure this
// package does not recognise.
const fallbackMessage = "does not satisfy the constraints defined in the schema"

// jsonTypeNames are the type names the jsonschema library can report. A name
// outside this set did not come from its type check, so it is not echoed.
var jsonTypeNames = map[string]struct{}{
	"null": {}, "boolean": {}, "number": {}, "integer": {},
	"string": {}, "array": {}, "object": {},
}

func dependencyDescription(prop string, missing []string) described {
	// Both the trigger property and the missing ones are declared by the schema's
	// dependentRequired map.
	names, ok := specTokens(missing)
	trigger, triggerOK := specToken(prop)
	if !ok || !triggerOK {
		return msg("is missing properties that the schema requires alongside the ones that were sent")
	}

	return msg("must also contain " + names + " when '" + trigger + "' is present")
}

func numericDescription(prefix string, want *big.Rat) described {
	if want == nil {
		return msg(fallbackMessage)
	}

	return msg(prefix + ratString(want))
}

// undeclaredDescription reports properties the schema does not define.
//
// The names that were sent cannot appear in the output, so the message says how
// many there were and, when one of them is a near miss for a property the schema
// does declare, which one that is. The suggestion is a declared name, so it comes
// from the specification even though the typo that prompted it did not.
func undeclaredDescription(sent []string, allowed propertyLookup) described {
	subject := "properties that are"
	switch len(sent) {
	case 0:
		// The library did not say how many, so stay vague rather than guess.
	case 1:
		subject = "1 property that is"
	default:
		subject = plural(len(sent), "property", "properties") + " that are"
	}

	if allowed != nil {
		if suggestion, ok := nearestDeclared(sent, allowed()); ok {
			return msgFix(
				"contains "+subject+" not defined in the schema. Did you mean '"+suggestion+"'?",
				"Rename it to '"+suggestion+"', or remove it if you did not mean to send it.")
		}
	}

	return msgFix("contains "+subject+" not defined in the schema", removeUndeclaredFix(allowed))
}

// removeUndeclaredFix tells the caller which properties the object does accept,
// since the ones it rejected are names they chose and cannot be repeated back.
func removeUndeclaredFix(allowed propertyLookup) string {
	if names, ok := declaredNames(allowed); ok {
		return "Remove any property that is not one of: " + names + "."
	}

	return "Remove any property this operation does not define. The names are withheld because they came from the request."
}

// renameUndeclaredFix is the same hint for a schema that constrains property
// names rather than listing them. A schema that does both is rare, so the list
// is usually unavailable and the fixed sentence is what goes out.
func renameUndeclaredFix(allowed propertyLookup) string {
	if names, ok := declaredNames(allowed); ok {
		return "Rename the property to one of: " + names + "."
	}

	return "Rename the property so that its name satisfies the schema. The name is withheld because it came from the request."
}

// declaredNames renders the accepted property names, sorted so the hint is
// stable, bounded in count the same way an enum list is, and with every name put
// through the length guard in case a future library version renders something
// other than a schema into the field these are read from.
func declaredNames(allowed propertyLookup) (string, bool) {
	if allowed == nil {
		return "", false
	}

	names := allowed()
	if len(names) == 0 {
		return "", false
	}

	kept := make([]string, 0, len(names))
	for _, name := range names {
		token, ok := specToken(name)
		if !ok {
			return "", false
		}
		kept = append(kept, token)
	}
	sort.Strings(kept)

	truncated := false
	if len(kept) > maxListedValues {
		kept = kept[:maxListedValues]
		truncated = true
	}

	rendered := strings.Join(kept, ", ")
	if truncated {
		rendered += ", …"
	}

	return rendered, true
}
