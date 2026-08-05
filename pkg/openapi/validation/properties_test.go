package validation

import (
	"net/http"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"testing"

	validatorErrors "github.com/pb33f/libopenapi-validator/errors"
	"github.com/stretchr/testify/require"
)

// The whole point of resolving the schema is that a caller who sent a property the
// operation does not define gets told what it does define, because the name they
// used is theirs and cannot be repeated back to them.
func TestValidateNamesTheAllowedProperties(t *testing.T) {
	t.Parallel()

	spec := []byte(`
openapi: 3.0.3
info: {title: probe, version: 1.0.0}
paths:
  /probe:
    post:
      requestBody:
        content:
          application/json:
            schema:
              type: object
              additionalProperties: false
              properties:
                name: {type: string}
                apiId: {type: string}
                nested:
                  type: object
                  additionalProperties: false
                  properties:
                    inner: {type: string}
      responses:
        "200": {description: ok}
`)

	v, err := NewFromBytes(spec)
	require.NoError(t, err)

	body := `{"` + canary + `":1,"nested":{"` + canary + `":2}}`
	r, err := http.NewRequest(http.MethodPost, "https://x/probe", strings.NewReader(body))
	require.NoError(t, err)
	r.Header.Set("Content-Type", "application/json")

	result := v.Validate(r)
	requireNoLeak(t, result, body)

	require.Equal(t, []ValidationError{
		{
			Location: "body",
			Message:  "contains 1 property that is not defined in the schema",
			Fix:      ptr("Remove any property that is not one of: apiId, name, nested."),
		},
		{
			Location: "body.nested",
			Message:  "contains 1 property that is not defined in the schema",
			Fix:      ptr("Remove any property that is not one of: inner."),
		},
	}, result.Errors)
}

// A schema the pointer does not resolve against, or one that declares no
// properties at all, has to leave the message actionable without the list.
func TestPropertyResolverDegradesGracefully(t *testing.T) {
	t.Parallel()

	cache := newPropertyCache()

	tests := []struct {
		name    string
		schema  string
		keyword string
		want    []string
	}{
		{
			name:    "a keyword at the root of the schema",
			schema:  "type: object\nproperties:\n  a: {type: string}\n  b: {type: string}\nadditionalProperties: false\n",
			keyword: "/additionalProperties",
			want:    []string{"a", "b"},
		},
		{
			name:    "a keyword nested under a property",
			schema:  "properties:\n  outer:\n    properties:\n      inner: {}\n    additionalProperties: false\n",
			keyword: "/properties/outer/additionalProperties",
			want:    []string{"inner"},
		},
		{
			name:    "an escaped segment",
			schema:  "properties:\n  \"a/b\":\n    properties:\n      inner: {}\n",
			keyword: "/properties/a~1b/additionalProperties",
			want:    []string{"inner"},
		},
		{
			name:    "a pointer that does not resolve",
			schema:  "type: object\nproperties:\n  a: {type: string}\n",
			keyword: "/properties/missing/additionalProperties",
			want:    nil,
		},
		{
			name:    "a schema with no properties map",
			schema:  "type: object\nadditionalProperties: false\n",
			keyword: "/additionalProperties",
			want:    nil,
		},
		{
			name:    "an unparseable schema",
			schema:  "\t: not: yaml: at: all\n",
			keyword: "/additionalProperties",
			want:    nil,
		},
		{
			name:    "an empty pointer",
			schema:  "type: object\nproperties:\n  a: {}\n",
			keyword: "",
			want:    nil,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			resolver := resolverFor(cache, tt.schema)
			got := resolver.lookup(tt.keyword)
			if tt.want == nil {
				require.Empty(t, got)

				return
			}
			require.ElementsMatch(t, tt.want, got)
		})
	}
}

// Nothing to resolve is the common case: most failures never ask, and a library
// version that stops handing over the rendered schema must not panic.
func TestPropertyResolverWithNothingToResolve(t *testing.T) {
	t.Parallel()

	require.Nil(t, newPropertyResolver(newPropertyCache(), &validationError{}).lookup("/additionalProperties"))
	require.Nil(t, resolverFor(nil, "properties: {a: {}}").lookup("/additionalProperties"))

	var absent *propertyResolver
	require.Nil(t, absent.lookup("/additionalProperties"))
}

// The memo is what keeps a repeated rejection from re-parsing the schema, and the
// bound is what keeps a long-lived frontline validator from growing without limit.
func TestPropertyCacheMemoisesAndEvicts(t *testing.T) {
	t.Parallel()

	cache := newPropertyCache()
	const schema = "properties:\n  a: {}\n  b: {}\n"

	first := resolverFor(cache, schema)
	require.ElementsMatch(t, []string{"a", "b"}, first.lookup("/additionalProperties"))

	// A second request renders the same schema again, and answers without parsing.
	second := resolverFor(cache, schema)
	require.ElementsMatch(t, []string{"a", "b"}, second.lookup("/additionalProperties"))
	require.False(t, second.done, "the schema was parsed again on a cache hit")

	// A different pointer into the same schema is a different entry.
	require.Empty(t, second.lookup("/properties/a/additionalProperties"))
	require.True(t, second.done)

	for i := range maxPropertyCacheEntries * 2 {
		resolverFor(cache, schema).lookup("/pointer/" + strconv.Itoa(i))
	}
	require.LessOrEqual(t, len(cache.entries), maxPropertyCacheEntries)
	require.Len(t, cache.order, len(cache.entries))
}

// Two specifications that happen to be resolved by the same validator would share
// a cache, so the key has to separate them. In practice each compiled
// specification has its own, which is what keeps one customer's property names out
// of another customer's response.
func TestPropertyCacheSeparatesSchemas(t *testing.T) {
	t.Parallel()

	cache := newPropertyCache()

	require.ElementsMatch(t, []string{"a"},
		resolverFor(cache, "properties:\n  a: {}\n").lookup("/additionalProperties"))
	require.ElementsMatch(t, []string{"b"},
		resolverFor(cache, "properties:\n  b: {}\n").lookup("/additionalProperties"))
}

// resolverFor builds a resolver over a schema string, the way the library hands
// one over on a failure.
func resolverFor(cache *propertyCache, schema string) *propertyResolver {
	return newPropertyResolver(cache, &validationError{
		SchemaValidationErrors: []*validatorErrors.SchemaValidationFailure{{ReferenceSchema: schema}},
	})
}

// BenchmarkDeclaredProperties measures the two paths of the memo against the real
// keys.createKey schema, which is the largest request body this API declares. A
// miss parses the rendered schema; a hit hashes it and reads a map. The rejection
// path is the only one that pays either.
func BenchmarkDeclaredProperties(b *testing.B) {
	schema := renderedCreateKeySchema(b)

	b.Run("miss", func(b *testing.B) {
		b.ReportAllocs()
		for range b.N {
			cache := newPropertyCache()
			names := resolverFor(cache, schema).lookup("/additionalProperties")
			if len(names) == 0 {
				b.Fatal("expected the schema to declare properties")
			}
		}
	})

	b.Run("hit", func(b *testing.B) {
		cache := newPropertyCache()
		if len(resolverFor(cache, schema).lookup("/additionalProperties")) == 0 {
			b.Fatal("expected the schema to declare properties")
		}

		b.ReportAllocs()
		b.ResetTimer()
		for range b.N {
			if len(resolverFor(cache, schema).lookup("/additionalProperties")) == 0 {
				b.Fatal("expected the schema to declare properties")
			}
		}
	})
}

// renderedCreateKeySchema takes the rendered schema the library produces for a
// rejected keys.createKey body. It is read off a real failure rather than written
// out here, because the shape of that string is the library's business and the
// point of the benchmark is what it actually costs to parse.
func renderedCreateKeySchema(b *testing.B) string {
	b.Helper()

	// A benchmark against the real specification, so it is read from the tree
	// rather than imported: pkg must not depend on svc.
	path := filepath.Join("..", "..", "..", "svc", "api", "openapi", "openapi-generated.yaml")
	spec, err := os.ReadFile(path) //nolint:gosec // fixed path inside the repository
	if err != nil {
		b.Skipf("the API specification is not readable from here: %v", err)
	}

	v, err := NewFromBytes(spec)
	require.NoError(b, err)

	//nolint:noctx // no context available in a benchmark helper
	r, err := http.NewRequest(http.MethodPost, "https://x/v2/keys.createKey",
		strings.NewReader(`{"apiId":"api_abc","surprise":1}`))
	require.NoError(b, err)
	r.Header.Set("Content-Type", "application/json")
	r.Header.Set("Authorization", "Bearer unkey_test")

	_, errs := v.validator.ValidateHttpRequestSync(r)
	for _, e := range errs {
		for _, failure := range e.SchemaValidationErrors {
			if failure != nil && failure.ReferenceSchema != "" {
				b.Logf("rendered keys.createKey schema: %d bytes", len(failure.ReferenceSchema))

				return failure.ReferenceSchema
			}
		}
	}
	b.Fatal("the validator did not hand over a rendered schema")

	return ""
}
