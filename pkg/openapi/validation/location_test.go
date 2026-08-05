package validation

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi-validator/helpers"
	"github.com/stretchr/testify/require"
)

func TestRenderLocation(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name     string
		prefix   string
		instance string
		keyword  string
		want     string
	}{
		{
			name:   "an error on the whole body is the prefix alone",
			prefix: locationBody, instance: "", keyword: "/required",
			want: "body",
		},
		{
			name:   "a declared property",
			prefix: locationBody, instance: "/name", keyword: "/properties/name/minLength",
			want: "body.name",
		},
		{
			name:   "an array index",
			prefix: locationBody, instance: "/tags/3", keyword: "/properties/tags/items/type",
			want: "body.tags[3]",
		},
		{
			name:   "nested declared properties through an array",
			prefix: locationBody, instance: "/variables/1/key",
			keyword: "/properties/variables/items/properties/key/pattern",
			want:    "body.variables[1].key",
		},
		{
			name:   "a caller-chosen key under a free-form object is redacted",
			prefix: locationBody, instance: "/meta/" + canary,
			keyword: "/properties/meta/additionalProperties/type",
			want:    "body.meta.*",
		},
		{
			name:   "a declared property below a redacted segment survives",
			prefix: locationBody, instance: "/free/" + canary + "/inner",
			keyword: "/properties/free/additionalProperties/properties/inner/type",
			want:    "body.free.*.inner",
		},
		{
			name:   "the name from an additionalProperties failure is never in the instance path",
			prefix: locationBody, instance: "", keyword: "/additionalProperties",
			want: "body",
		},
		{
			name:   "a property name that looks like a keyword is still checked",
			prefix: locationBody, instance: "/pattern", keyword: "/properties/pattern/type",
			want: "body.pattern",
		},
		{
			name:   "a numeric property name declared by the schema is not an index",
			prefix: locationBody, instance: "/2024", keyword: "/properties/2024/type",
			want: "body.2024",
		},
		{
			name:   "an escaped segment is decoded once it is known to be declared",
			prefix: locationBody, instance: "/a~1b", keyword: "/properties/a~1b/type",
			want: "body.a/b",
		},
		{
			name:   "a parameter prefix carries the whole location",
			prefix: "query.filter", instance: "/from", keyword: "/properties/from/format",
			want: "query.filter.from",
		},
		{
			name:   "a segment that is neither declared nor an index is redacted",
			prefix: locationBody, instance: "/" + canary, keyword: "/propertyNames/pattern",
			want: "body.*",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := renderLocation(tt.prefix, tt.instance, tt.keyword)
			require.Equal(t, tt.want, got)
			require.NotContains(t, got, canary)
		})
	}
}

// The schema pointer is the only thing that makes a property name safe to echo,
// so a pointer that does not declare the name must never let it through, however
// the name is spelled.
func TestRenderLocationRedactsUndeclaredNames(t *testing.T) {
	t.Parallel()

	keywords := []string{
		"",
		"/",
		"/additionalProperties",
		"/additionalProperties/type",
		"/patternProperties/^x-/type",
		"/propertyNames",
		"/properties",
		"/properties/other/type",
		"/items/type",
	}

	for _, keyword := range keywords {
		got := renderLocation(locationBody, "/"+canary, keyword)
		require.Equal(t, "body.*", got, "keyword location %q", keyword)
	}
}

// The parameter name in a location comes from the specification's parameter list,
// but it still goes through the length guard, so a future code path that fills the
// field from the request loses the name instead of echoing it.
func TestPrefixForBoundsTheParameterName(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  validationError
		want string
	}{
		{
			name: "a declared header parameter",
			err: validationError{
				ValidationType:    helpers.ParameterValidation,
				ValidationSubType: helpers.ParameterValidationHeader,
				ParameterName:     "X-Secret",
			},
			want: "header.X-Secret",
		},
		{
			name: "a name long enough to be a payload",
			err: validationError{
				ValidationType:    helpers.ParameterValidation,
				ValidationSubType: helpers.ParameterValidationQuery,
				ParameterName:     strings.Repeat("a", maxSpecTokenLen+1),
			},
			want: "query",
		},
		{
			name: "a body content type failure points at the header it read",
			err: validationError{
				ValidationType:    helpers.RequestBodyValidation,
				ValidationSubType: helpers.RequestBodyContentType,
			},
			want: "header.Content-Type",
		},
		{
			name: "an http security scheme points at the Authorization header",
			err: validationError{
				ValidationType:    helpers.SecurityValidation,
				ValidationSubType: "bearer",
			},
			want: "header.Authorization",
		},
		{
			name: "an apiKey scheme names its own carrier, which is not recoverable here",
			err: validationError{
				ValidationType:    helpers.SecurityValidation,
				ValidationSubType: apiKeySubType,
			},
			want: locationRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := prefixFor(&tt.err)
			require.Equal(t, tt.want, got)
		})
	}
}
