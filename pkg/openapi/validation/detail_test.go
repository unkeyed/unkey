package validation

import (
	"strings"
	"testing"

	"github.com/pb33f/libopenapi-validator/helpers"
	"github.com/stretchr/testify/require"
)

func TestDetailFor(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		err  validationError
		want string
	}{
		{
			name: "a body schema failure keeps the wording the API has always returned",
			err: validationError{
				ValidationType:    helpers.RequestBodyValidation,
				ValidationSubType: helpers.Schema,
				RequestMethod:     "POST",
				SpecPath:          "/v2/keys.verifyKey",
				RequestPath:       "/v2/keys.verifyKey",
				Message:           "POST request body for '/v2/keys.verifyKey' failed to validate schema",
				Reason:            "The request body is defined as an object. However, it does not meet the schema requirements of the specification",
			},
			want: "POST request body for '/v2/keys.verifyKey' failed to validate schema",
		},
		{
			// The library builds its own message from the request's URL path, which in
			// a customer specification can be the credential. The path template is what
			// goes out instead.
			name: "the path in the summary comes from the specification, not the request",
			err: validationError{
				ValidationType:    helpers.RequestBodyValidation,
				ValidationSubType: helpers.Schema,
				RequestMethod:     "POST",
				SpecPath:          "/reset/{token}",
				RequestPath:       "/reset/" + canary,
				Message:           "POST request body for '/reset/" + canary + "' failed to validate schema",
			},
			want: "POST request body for '/reset/{token}' failed to validate schema",
		},
		{
			name: "an unparseable body is still a schema failure at this level",
			err: validationError{
				ValidationType:    helpers.RequestBodyValidation,
				ValidationSubType: helpers.Schema,
				RequestMethod:     "POST",
				SpecPath:          "/v2/apis.createApi",
				Reason:            reasonBodyUndecodable + ": invalid character '" + canary + "'",
			},
			want: "POST request body for '/v2/apis.createApi' failed to validate schema",
		},
		{
			name: "a missing body",
			err: validationError{
				ValidationType:    helpers.RequestBodyValidation,
				ValidationSubType: helpers.Schema,
				RequestMethod:     "POST",
				SpecPath:          "/v2/apis.createApi",
				Reason:            reasonBodyEmpty,
			},
			want: "POST request body is empty for '/v2/apis.createApi'",
		},
		{
			name: "an unsupported content type does not name the type that was sent",
			err: validationError{
				ValidationType:    helpers.RequestBodyValidation,
				ValidationSubType: helpers.RequestBodyContentType,
				RequestMethod:     "POST",
				SpecPath:          "/v2/keys.verifyKey",
				Message:           "POST operation request content type 'text/plain; boundary=" + canary + "' does not exist",
			},
			want: "POST operation for '/v2/keys.verifyKey' does not accept the request content type",
		},
		{
			name: "an unknown method degrades to the generic summary",
			err: validationError{
				ValidationType:    helpers.RequestBodyValidation,
				ValidationSubType: helpers.Schema,
				RequestMethod:     canary,
				SpecPath:          "/v2/keys.verifyKey",
			},
			want: genericDetail,
		},
		{
			name: "a parameter failure is rebuilt from its name and its problem",
			err: validationError{
				ValidationType:    helpers.ParameterValidation,
				ValidationSubType: helpers.ParameterValidationQuery,
				ParameterName:     "limit",
				RequestMethod:     "GET",
				Message:           "Query parameter 'limit' is not a valid integer",
			},
			want: "Query parameter 'limit' must be an integer",
		},
		{
			name: "an array parameter keeps the word array",
			err: validationError{
				ValidationType:    helpers.ParameterValidation,
				ValidationSubType: helpers.ParameterValidationQuery,
				ParameterName:     "tags",
				Message:           "Query array parameter 'tags' has too many items",
			},
			want: "Query array parameter 'tags' contains more items than the schema allows",
		},
		{
			name: "a parameter message this package has not seen degrades to fixed text",
			err: validationError{
				ValidationType:    helpers.ParameterValidation,
				ValidationSubType: helpers.ParameterValidationHeader,
				ParameterName:     "X-Secret",
				Message:           "Header parameter 'X-Secret' was rejected because '" + canary + "' is wrong",
			},
			want: "Header parameter 'X-Secret' failed validation",
		},
		{
			name: "a parameter message whose entity disagrees with the error is not trusted",
			err: validationError{
				ValidationType:    helpers.ParameterValidation,
				ValidationSubType: helpers.ParameterValidationHeader,
				ParameterName:     "X-Secret",
				Message:           "Query parameter 'X-Secret' is missing",
			},
			want: "Header parameter 'X-Secret' failed validation",
		},
		{
			name: "a parameter with no declared name degrades to the generic summary",
			err: validationError{
				ValidationType:    helpers.ParameterValidation,
				ValidationSubType: helpers.ParameterValidationQuery,
				ParameterName:     "",
				Message:           "Query parameter '" + canary + "' is missing",
			},
			want: genericDetail,
		},
		{
			name: "a missing credential",
			err: validationError{
				ValidationType:    helpers.SecurityValidation,
				ValidationSubType: "bearer",
				Message:           "Authorization header for 'bearer' scheme",
				Reason:            "Authorization header was not found",
			},
			want: "Authorization header for 'bearer' scheme",
		},
		{
			name: "a missing security scheme",
			err: validationError{
				ValidationType: helpers.SecurityValidation,
				Message:        "Security scheme 'rootKey' is missing",
			},
			want: "Security scheme 'rootKey' is missing",
		},
		{
			name: "a security message this package has not seen",
			err: validationError{
				ValidationType: helpers.SecurityValidation,
				Message:        "Authorization value '" + canary + "' was rejected",
			},
			want: "The request is not authorized",
		},
		{
			name: "a path that is not in the specification does not repeat the path",
			err: validationError{
				ValidationType:    helpers.PathValidation,
				ValidationSubType: helpers.ValidationMissing,
				RequestMethod:     "POST",
				RequestPath:       "/reset/" + canary,
				Message:           "POST Path '/reset/" + canary + "' not found",
			},
			want: "POST request does not match any path and operation in the specification",
		},
		{
			name: "an unrecognised category",
			err: validationError{
				ValidationType: "somethingNew",
				Message:        canary,
				Reason:         canary,
			},
			want: genericDetail,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			got := detailFor(&tt.err)
			require.Equal(t, tt.want, got)
			require.NotContains(t, got, canary)
		})
	}
}

// The parameter message is taken apart rather than pattern-matched: both halves
// have to be known before anything from it is emitted, so a reworded or extended
// message loses detail rather than forwarding whatever the library now says.
func TestSplitParameterMessage(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name      string
		message   string
		parameter string
		word      string
		entity    string
		predicate string
		ok        bool
	}{
		{
			name: "a known entity and a known problem", message: "Query parameter 'limit' is missing",
			parameter: "limit", word: "Query",
			entity: "Query parameter", predicate: "is required but was not provided", ok: true,
		},
		{
			name: "the array variant", message: "Header array parameter 'tags' is not a valid number",
			parameter: "tags", word: "Header",
			entity: "Header array parameter", predicate: "must be a number", ok: true,
		},
		{
			name:      "an unknown problem",
			message:   "Query parameter 'limit' exploded in a brand new way",
			parameter: "limit", word: "Query",
		},
		{
			name:      "an unknown entity",
			message:   "Query thing 'limit' is missing",
			parameter: "limit", word: "Query",
		},
		{
			name:      "an entity that disagrees with the structured location",
			message:   "Query parameter 'limit' is missing",
			parameter: "limit", word: "Header",
		},
		{
			name:      "a message the name does not appear in",
			message:   "Query parameter is missing",
			parameter: "limit", word: "Query",
		},
		{
			name:      "prose smuggled in ahead of the name",
			message:   "Query parameter '" + canary + "' and Query parameter 'limit' is missing",
			parameter: "limit", word: "Query",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			entity, predicate, ok := splitParameterMessage(tt.message, tt.parameter, tt.word)
			require.Equal(t, tt.ok, ok)
			require.Equal(t, tt.entity, entity)
			require.Equal(t, tt.predicate, predicate)
			require.NotContains(t, entity+predicate, canary)
		})
	}
}

// Every problem phrase the library can write has to be in the table, or a real
// failure degrades to "failed validation" for no reason. This pins the mapping so
// that the set is reviewed rather than assumed.
func TestParameterProblemsAreMappedToPredicates(t *testing.T) {
	t.Parallel()

	for problem, predicate := range parameterProblems {
		require.NotEmpty(t, predicate, "problem %q has no predicate", problem)
		require.NotEqual(t, problem, predicate,
			"problem %q is forwarded rather than rewritten", problem)
	}
}

func TestSupportedContentTypes(t *testing.T) {
	t.Parallel()

	require.Equal(t, "application/json",
		supportedContentTypes("The content type is invalid, Use one of the 1 supported types for this operation: application/json"))
	require.Equal(t, "application/json, application/xml",
		supportedContentTypes("The content type is invalid, Use one of the 2 supported types for this operation: application/json, application/xml"))
	require.Empty(t, supportedContentTypes("Ensure that the object being submitted, matches the schema correctly"))
	require.Empty(t, supportedContentTypes("The content type '"+canary+"' is invalid"))
	require.Empty(t, supportedContentTypes(
		"The content type is invalid, Use one of the 1 supported types for this operation: "+
			strings.Repeat("a", maxSpecTokenLen+1)))
}
