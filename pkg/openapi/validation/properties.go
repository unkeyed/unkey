package validation

import (
	"hash/maphash"
	"strconv"
	"sync"

	"gopkg.in/yaml.v3"
)

// When a request carries a property the schema does not declare, the only thing
// this package can tell the caller is what the schema does declare. The name they
// used is theirs, and a caller-chosen property name is as likely to hold a
// credential as a value is, so it is never echoed.
//
// libopenapi-validator hands over the rendered schema that did the rejecting,
// with $refs already expanded, and a JSON Pointer to the keyword that failed.
// Resolving that pointer's parent and reading its properties map gives the
// accepted names, straight from the specification.
//
// The rendered schema is a string, so reading it means parsing it, and for a
// schema the size of keys.createKey that is not free. Three things keep the cost
// off the healthy path and off the repeated rejection:
//
//   - It is only resolved when an additionalProperties or propertyNames keyword
//     actually failed, and only for the failing pointer.
//   - Within one request the document is parsed at most once, however many
//     properties were rejected.
//   - Across requests the resolved names are memoised in a bounded cache on the
//     Validator, keyed by the identity of the rendered schema and the failing
//     pointer, so the steady state for a spec that keeps being sent the same bad
//     property is a hash and a map lookup. See BenchmarkDeclaredProperties for
//     what each of the two paths costs.

// propertyLookup returns the property names a schema declares next to the
// keyword that failed, or nil when they cannot be resolved.
type propertyLookup func() []string

// maxPropertyCacheEntries bounds the memo. Entries are keyed by pointer as well
// as by schema, so an operation with objects at many depths takes several, and a
// frontline instance holds one cache per compiled customer specification.
const maxPropertyCacheEntries = 256

// propertyCacheKey identifies one resolution: which rendered schema, and which
// keyword inside it.
//
// The schema is identified by a hash of its text rather than by the text itself,
// because the library renders a fresh string per failure and the schema for a
// large operation is tens of kilobytes. The hash is seeded per cache, so a
// collision cannot be arranged from outside, and a collision would at worst
// substitute the property names of another subschema of the same specification:
// the cache lives on the Validator rather than in a package variable, so it
// cannot mix one customer's specification into another's response.
type propertyCacheKey struct {
	schema  uint64
	keyword string
}

// propertyCache is a bounded, concurrency-safe memo of resolved property names.
// Eviction is first-in-first-out, which is enough for a fixed set of schemas:
// there is nothing to age out, only a ceiling to respect.
type propertyCache struct {
	mu      sync.Mutex
	seed    maphash.Seed
	entries map[propertyCacheKey][]string
	order   []propertyCacheKey
}

func newPropertyCache() *propertyCache {
	return &propertyCache{
		mu:      sync.Mutex{},
		seed:    maphash.MakeSeed(),
		entries: make(map[propertyCacheKey][]string, maxPropertyCacheEntries),
		order:   make([]propertyCacheKey, 0, maxPropertyCacheEntries),
	}
}

func (c *propertyCache) get(key propertyCacheKey) ([]string, bool) {
	c.mu.Lock()
	defer c.mu.Unlock()

	names, ok := c.entries[key]

	return names, ok
}

func (c *propertyCache) put(key propertyCacheKey, names []string) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if _, exists := c.entries[key]; exists {
		return
	}

	if len(c.order) >= maxPropertyCacheEntries {
		oldest := c.order[0]
		c.order = c.order[1:]
		delete(c.entries, oldest)
	}

	c.entries[key] = names
	c.order = append(c.order, key)
}

// propertyResolver resolves declared property names for one top-level validation
// error. It is created per request and is not safe for concurrent use, which it
// does not need to be: one request is translated on one goroutine.
type propertyResolver struct {
	raw    string
	hash   uint64
	cache  *propertyCache
	parsed any
	done   bool
}

// newPropertyResolver takes the rendered schema off the first failure that
// carries one. Every failure the library flattened out of the same schema
// reports the same rendered text.
func newPropertyResolver(cache *propertyCache, e *validationError) *propertyResolver {
	raw := ""
	for _, failure := range e.SchemaValidationErrors {
		if failure != nil && failure.ReferenceSchema != "" {
			raw = failure.ReferenceSchema

			break
		}
	}

	resolver := &propertyResolver{
		raw:    raw,
		hash:   0,
		cache:  cache,
		parsed: nil,
		done:   false,
	}
	if raw != "" && cache != nil {
		resolver.hash = maphash.String(cache.seed, raw)
	}

	return resolver
}

// lookup returns the property names declared alongside the keyword at
// keywordLocation. The rendered schema is YAML for request bodies and JSON for
// parameters, and YAML parses both.
func (r *propertyResolver) lookup(keywordLocation string) []string {
	if r == nil || r.raw == "" || r.cache == nil {
		return nil
	}

	key := propertyCacheKey{schema: r.hash, keyword: keywordLocation}
	if names, ok := r.cache.get(key); ok {
		return names
	}

	names := r.resolve(keywordLocation)
	r.cache.put(key, names)

	return names
}

func (r *propertyResolver) resolve(keywordLocation string) []string {
	if !r.done {
		r.done = true
		var parsed any
		if err := yaml.Unmarshal([]byte(r.raw), &parsed); err == nil {
			r.parsed = parsed
		}
	}
	if r.parsed == nil {
		return nil
	}

	segments := pointerSegments(keywordLocation)
	if len(segments) == 0 {
		return nil
	}
	for i, segment := range segments {
		segments[i] = unescapePointerSegment(segment)
	}

	// The keyword's parent is the schema object that declares the properties.
	node, ok := walkPointer(r.parsed, segments[:len(segments)-1])
	if !ok {
		return nil
	}

	object, ok := node.(map[string]any)
	if !ok {
		return nil
	}

	properties, ok := object["properties"].(map[string]any)
	if !ok {
		return nil
	}

	names := make([]string, 0, len(properties))
	for name := range properties {
		names = append(names, name)
	}

	return names
}

// walkPointer follows JSON Pointer segments through a decoded document.
func walkPointer(node any, segments []string) (any, bool) {
	for _, segment := range segments {
		switch current := node.(type) {
		case map[string]any:
			next, ok := current[segment]
			if !ok {
				return nil, false
			}
			node = next

		case []any:
			index, err := strconv.Atoi(segment)
			if err != nil || index < 0 || index >= len(current) {
				return nil, false
			}
			node = current[index]

		default:
			return nil, false
		}
	}

	return node, true
}
