package redaction

import (
	"sort"
	"strings"

	"gopkg.in/yaml.v3"

	"github.com/unkeyed/unkey/pkg/fault"
)

// Extension is the OpenAPI vendor extension marking a schema property whose
// value must never reach request logs:
//
//	value:
//	  type: string
//	  x-unkey-redact: true
const Extension = "x-unkey-redact"

// schemaRefPrefix is the only $ref form the bundled spec contains.
const schemaRefPrefix = "#/components/schemas/"

// PathsFromSpec returns the sorted body paths of every schema property in spec
// annotated with [Extension], in the form [New] accepts.
//
// Paths are anchored at the root of a request or response body, so one annotated
// property yields one path per place it is reachable. Annotating the environment
// variable value produces both of these, since the request sends variables at the
// root and the response nests them under data:
//
//	variables[].value
//	data.variables[].value
//
// Walking operations rather than schema components is what makes the paths
// meaningful, since a component on its own has no root to be relative to. $refs
// are resolved along the way, and a schema that refers back to itself stops at
// the repeat instead of recursing forever.
func PathsFromSpec(spec []byte) ([]string, error) {
	var doc map[string]any
	if err := yaml.Unmarshal(spec, &doc); err != nil {
		return nil, fault.Wrap(err, fault.Internal("failed to parse OpenAPI spec for redaction paths"))
	}

	w := &walker{
		schemas:       mapAt(mapAt(doc, "components"), "schemas"),
		found:         make(map[string]struct{}),
		truncated:     make(map[string]struct{}),
		annotated:     make(map[string]bool),
		unaddressable: nil,
	}

	for _, item := range mapAt(doc, "paths") {
		operations := toMap(item)
		for _, operation := range operations {
			op := toMap(operation)
			if op == nil {
				continue
			}

			request := bodySchema(mapAt(op, "requestBody"))
			w.flagUnaddressable(request, nil, "a request body root")
			w.walk(request, nil, nil)

			for _, response := range mapAt(op, "responses") {
				body := bodySchema(toMap(response))
				w.flagUnaddressable(body, nil, "a response body root")
				w.walk(body, nil, nil)
			}
		}
	}

	// A schema that reaches itself has an annotated property at every depth, and
	// a finite path set cannot express that. Rather than emit paths that cover
	// the first level and quietly miss the rest, refuse: the caller fails
	// startup, and whoever added the recursive schema finds out immediately.
	if len(w.truncated) > 0 {
		cycles := make([]string, 0, len(w.truncated))
		for name := range w.truncated {
			cycles = append(cycles, name)
		}
		sort.Strings(cycles)

		return nil, fault.New("recursive schema with a redacted property cannot be expressed as paths: " + strings.Join(cycles, ", "))
	}

	if len(w.unaddressable) > 0 {
		sort.Strings(w.unaddressable)

		return nil, fault.New("only a property can carry " + Extension + ", found it on: " + strings.Join(w.unaddressable, ", "))
	}

	paths := make([]string, 0, len(w.found))
	for p := range w.found {
		paths = append(paths, p)
	}
	sort.Strings(paths)

	return paths, nil
}

// walker collects annotated paths across every operation in one spec.
type walker struct {
	schemas map[string]any
	found   map[string]struct{}

	// truncated names components whose walk stopped at a cycle while something
	// annotated sits inside them, which is the one case paths cannot express.
	truncated map[string]struct{}

	// unaddressable collects annotations on schemas that are not properties, so
	// they fail loudly instead of doing nothing.
	unaddressable []string

	// annotated memoizes whether a component's subtree contains an annotation.
	annotated map[string]bool
}

// walk descends schema, recording the path of every annotated property. The
// visiting set holds the $refs already open on this descent, so a
// self-referential schema terminates.
func (w *walker) walk(schema map[string]any, path []string, visiting map[string]bool) {
	schema, visiting, ok := w.resolve(schema, visiting)
	if !ok {
		return
	}

	for name, raw := range mapAt(schema, "properties") {
		property := toMap(raw)
		if property == nil {
			continue
		}

		// Read the annotation before resolving so it can sit next to a $ref on
		// the property itself, then after, so it is also found on the component
		// the property points at.
		annotatedHere := annotated(property)
		if !annotatedHere {
			if resolved, _, ok := w.resolve(property, visiting); ok {
				annotatedHere = annotated(resolved)
			}
		}
		if annotatedHere {
			w.found[strings.Join(extend(path, name), ".")] = struct{}{}
		}

		w.walk(property, extend(path, name), visiting)
	}

	// Array elements extend the last segment rather than adding one, so
	// variables plus items becomes variables[]. A body that is itself an array
	// has no last segment to extend, and its elements are addressed by a leading
	// step instead: [].identifier. /v2/ratelimit.multiLimit is exactly this
	// shape, and skipping it would silently leave that operation unredacted while
	// its object-bodied twin /v2/ratelimit.limit was covered.
	if items := toMap(schema["items"]); items != nil {
		element := []string{"[]"}
		if len(path) > 0 {
			element = extend(path[:len(path)-1], path[len(path)-1]+"[]")
		}
		w.flagUnaddressable(items, visiting, strings.Join(element, ".")+" elements")
		w.walk(items, element, visiting)
	}

	// Composition keeps the same path: every branch describes the same object.
	for _, keyword := range []string{"allOf", "anyOf", "oneOf"} {
		branches, ok := schema[keyword].([]any)
		if !ok {
			continue
		}
		for _, branch := range branches {
			if b := toMap(branch); b != nil {
				w.walk(b, path, visiting)
			}
		}
	}
}

// flagUnaddressable records an annotation on a schema that is not a property.
// Only a property has a name to build a path from, so an annotation on a body root
// or on an array's items schema marks a value the path syntax cannot name. Doing
// nothing about it would mean an author marks a secret and it is never redacted,
// which is the failure this whole mechanism exists to prevent.
func (w *walker) flagUnaddressable(schema map[string]any, visiting map[string]bool, where string) {
	if schema == nil {
		return
	}
	if annotated(schema) {
		w.unaddressable = append(w.unaddressable, where)

		return
	}
	if resolved, _, ok := w.resolve(schema, visiting); ok && annotated(resolved) {
		w.unaddressable = append(w.unaddressable, where)
	}
}

// resolve follows a $ref to its component, returning the visiting set extended
// with that ref. It reports false when the ref is unresolvable or already open,
// which is how self-referential schemas terminate.
func (w *walker) resolve(schema map[string]any, visiting map[string]bool) (map[string]any, map[string]bool, bool) {
	if schema == nil {
		return nil, visiting, false
	}

	ref, isRef := schema["$ref"].(string)
	if !isRef {
		return schema, visiting, true
	}

	name, found := strings.CutPrefix(ref, schemaRefPrefix)
	if !found {
		return nil, visiting, false
	}
	if visiting[name] {
		if w.hasAnnotation(name, make(map[string]bool)) {
			w.truncated[name] = struct{}{}
		}

		return nil, visiting, false
	}

	target := toMap(w.schemas[name])
	if target == nil {
		return nil, visiting, false
	}

	// Copy rather than mutate, so sibling properties do not see each other's
	// refs as already open.
	extended := make(map[string]bool, len(visiting)+1)
	for k := range visiting {
		extended[k] = true
	}
	extended[name] = true

	return target, extended, true
}

// hasAnnotation reports whether the named component's subtree contains an
// annotated property, following refs. Results are memoized; a ref already open on
// the current descent contributes nothing, which keeps cycles finite.
func (w *walker) hasAnnotation(name string, open map[string]bool) bool {
	if cached, ok := w.annotated[name]; ok {
		return cached
	}
	if open[name] {
		return false
	}
	open[name] = true

	result := w.schemaHasAnnotation(toMap(w.schemas[name]), open)
	w.annotated[name] = result

	return result
}

// schemaHasAnnotation reports whether schema or anything beneath it is annotated.
func (w *walker) schemaHasAnnotation(schema map[string]any, open map[string]bool) bool {
	if schema == nil {
		return false
	}

	if ref, ok := schema["$ref"].(string); ok {
		target, found := strings.CutPrefix(ref, schemaRefPrefix)
		if !found {
			return false
		}

		return w.hasAnnotation(target, open)
	}

	for _, raw := range mapAt(schema, "properties") {
		property := toMap(raw)
		if property == nil {
			continue
		}
		if annotated(property) || w.schemaHasAnnotation(property, open) {
			return true
		}
	}

	if w.schemaHasAnnotation(toMap(schema["items"]), open) {
		return true
	}

	for _, keyword := range []string{"allOf", "anyOf", "oneOf"} {
		branches, ok := schema[keyword].([]any)
		if !ok {
			continue
		}
		for _, branch := range branches {
			if w.schemaHasAnnotation(toMap(branch), open) {
				return true
			}
		}
	}

	return false
}

// extend returns path with segment appended, without ever aliasing path's
// backing array. Sibling properties each build their own path from a shared
// prefix, so append alone would let one overwrite another's last segment.
func extend(path []string, segment string) []string {
	out := make([]string, len(path)+1)
	copy(out, path)
	out[len(path)] = segment

	return out
}

// bodySchema extracts the JSON schema from a requestBody or response object.
func bodySchema(container map[string]any) map[string]any {
	for mediaType, raw := range mapAt(container, "content") {
		if !strings.Contains(mediaType, "json") {
			continue
		}
		if media := toMap(raw); media != nil {
			return toMap(media["schema"])
		}
	}

	return nil
}

// annotated reports whether schema sets the annotation to true. A non-boolean
// value is ignored rather than treated as truthy, so a typo in the spec shows up
// as a missing path in the pinned list instead of silent redaction.
func annotated(schema map[string]any) bool {
	enabled, ok := schema[Extension].(bool)

	return ok && enabled
}

// mapAt returns the map stored at key, or nil.
func mapAt(m map[string]any, key string) map[string]any {
	if m == nil {
		return nil
	}

	return toMap(m[key])
}

// toMap returns v as a map, or nil when it is anything else.
func toMap(v any) map[string]any {
	m, ok := v.(map[string]any)
	if !ok {
		return nil
	}

	return m
}
