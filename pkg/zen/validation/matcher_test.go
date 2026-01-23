package validation

import (
	"testing"

	"github.com/stretchr/testify/require"
)

func TestPathMatcher(t *testing.T) {
	ops := map[string]*Operation{
		"POST /v2/keys.setRoles": {
			Method:      "POST",
			Path:        "/v2/keys.setRoles",
			OperationID: "keys.setRoles",
			Parameters:  ParameterSet{},
			Security:    nil,
		},
	}

	matcher := NewPathMatcher(ops)

	result, found := matcher.Match("POST", "/v2/keys.setRoles")
	require.True(t, found)
	require.Equal(t, "keys.setRoles", result.Operation.OperationID)

	result, found = matcher.Match("post", "/v2/keys.setRoles") // lowercase
	require.True(t, found)
	require.Equal(t, "keys.setRoles", result.Operation.OperationID)

	_, found = matcher.Match("GET", "/v2/keys.setRoles")
	require.False(t, found)

	_, found = matcher.Match("POST", "/unknown")
	require.False(t, found)
}

func TestPathMatcher_TemplateMatching(t *testing.T) {
	ops := map[string]*Operation{
		"GET /users/{userId}": {
			Method:      "GET",
			Path:        "/users/{userId}",
			OperationID: "users.get",
			Parameters:  ParameterSet{},
			Security:    nil,
		},
		"GET /users/{userId}/posts/{postId}": {
			Method:      "GET",
			Path:        "/users/{userId}/posts/{postId}",
			OperationID: "users.posts.get",
			Parameters:  ParameterSet{},
			Security:    nil,
		},
		"POST /v2/keys.setRoles": {
			Method:      "POST",
			Path:        "/v2/keys.setRoles",
			OperationID: "keys.setRoles",
			Parameters:  ParameterSet{},
			Security:    nil,
		},
	}

	matcher := NewPathMatcher(ops)

	// Test exact match still works
	result, found := matcher.Match("POST", "/v2/keys.setRoles")
	require.True(t, found)
	require.Equal(t, "keys.setRoles", result.Operation.OperationID)
	require.Nil(t, result.PathParams)

	// Test single path parameter
	result, found = matcher.Match("GET", "/users/user_123")
	require.True(t, found)
	require.Equal(t, "users.get", result.Operation.OperationID)
	require.Equal(t, "user_123", result.PathParams["userId"])

	// Test multiple path parameters
	result, found = matcher.Match("GET", "/users/user_123/posts/post_456")
	require.True(t, found)
	require.Equal(t, "users.posts.get", result.Operation.OperationID)
	require.Equal(t, "user_123", result.PathParams["userId"])
	require.Equal(t, "post_456", result.PathParams["postId"])

	// Test no match
	_, found = matcher.Match("GET", "/unknown/path")
	require.False(t, found)

	// Test wrong method
	_, found = matcher.Match("POST", "/users/user_123")
	require.False(t, found)
}

func TestPathMatcher_SpecialCharactersInPath(t *testing.T) {
	ops := map[string]*Operation{
		"GET /api/v2/keys.getKey": {
			Method:      "GET",
			Path:        "/api/v2/keys.getKey",
			OperationID: "keys.getKey",
			Parameters:  ParameterSet{},
			Security:    nil,
		},
	}

	matcher := NewPathMatcher(ops)

	// Dots in paths should be matched literally
	result, found := matcher.Match("GET", "/api/v2/keys.getKey")
	require.True(t, found)
	require.Equal(t, "keys.getKey", result.Operation.OperationID)

	// Make sure the dot isn't treated as regex wildcard
	_, found = matcher.Match("GET", "/api/v2/keysXgetKey")
	require.False(t, found)
}
