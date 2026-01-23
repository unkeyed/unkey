//go:build benchmark_comparison

package validation

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/stretchr/testify/require"
)

// This file contains comparison benchmarks between the OLD (libopenapi) and NEW validator.
// It requires the benchmark_comparison build tag to be set.
//
// To run:
//   go test -bench=BenchmarkComparison -benchmem -tags=benchmark_comparison
//
// Note: The old validator requires the libopenapi dependencies to be available.

// BenchmarkComparison compares old vs new validator implementations
func BenchmarkComparison(b *testing.B) {
	// NEW validator benchmarks
	b.Run("new/init", benchNewInit)
	b.Run("new/validate_simple", benchNewValidateSimple)
	b.Run("new/validate_complex", benchNewValidateComplex)
	b.Run("new/parallel", benchNewParallel)

	// OLD validator benchmarks
	b.Run("old/init", benchOldInit)
	b.Run("old/validate_simple", benchOldValidateSimple)
	b.Run("old/validate_complex", benchOldValidateComplex)
	b.Run("old/parallel", benchOldParallel)
}

// --- NEW validator benchmarks ---

func benchNewInit(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := New()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchNewValidateSimple(b *testing.B) {
	v, err := New()
	if err != nil {
		b.Fatal(err)
	}

	body := `{"keyId": "key_123abc", "roles": ["admin"]}`

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test_key_abc123")
		_, _ = v.Validate(context.Background(), req)
	}
}

func benchNewValidateComplex(b *testing.B) {
	v, err := New()
	if err != nil {
		b.Fatal(err)
	}

	body := `{
		"keyId": "key_123abc",
		"roles": [
			{"roleId": "role_admin", "teamId": "team_1"},
			{"roleId": "role_user", "teamId": "team_2"},
			{"roleId": "role_viewer", "teamId": "team_3"}
		]
	}`

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test_key_abc123")
		_, _ = v.Validate(context.Background(), req)
	}
}

func benchNewParallel(b *testing.B) {
	v, err := New()
	if err != nil {
		b.Fatal(err)
	}

	body := `{"keyId": "key_123abc", "roles": ["admin", "user"]}`

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test_key_abc123")
			_, _ = v.Validate(context.Background(), req)
		}
	})
}

// --- OLD validator benchmarks ---

func benchOldInit(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_, err := NewOldValidator()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func benchOldValidateSimple(b *testing.B) {
	v, err := NewOldValidator()
	if err != nil {
		b.Fatal(err)
	}

	body := `{"keyId": "key_123abc", "roles": ["admin"]}`

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test_key_abc123")
		_ = v.ValidateRequest(req)
	}
}

func benchOldValidateComplex(b *testing.B) {
	v, err := NewOldValidator()
	if err != nil {
		b.Fatal(err)
	}

	body := `{
		"keyId": "key_123abc",
		"roles": [
			{"roleId": "role_admin", "teamId": "team_1"},
			{"roleId": "role_user", "teamId": "team_2"},
			{"roleId": "role_viewer", "teamId": "team_3"}
		]
	}`

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test_key_abc123")
		_ = v.ValidateRequest(req)
	}
}

func benchOldParallel(b *testing.B) {
	v, err := NewOldValidator()
	if err != nil {
		b.Fatal(err)
	}

	body := `{"keyId": "key_123abc", "roles": ["admin", "user"]}`

	b.ReportAllocs()
	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test_key_abc123")
			_ = v.ValidateRequest(req)
		}
	})
}

// BenchmarkComparisonMemory compares memory usage between old and new
func BenchmarkComparisonMemory(b *testing.B) {
	body := `{"keyId": "key_123abc", "roles": ["admin", "user", "viewer"]}`

	b.Run("new/memory_per_request", func(b *testing.B) {
		v, err := New()
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test_key_abc123")
			_, _ = v.Validate(context.Background(), req)
		}
	})

	b.Run("old/memory_per_request", func(b *testing.B) {
		v, err := NewOldValidator()
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test_key_abc123")
			_ = v.ValidateRequest(req)
		}
	})
}

// BenchmarkComparisonThroughput compares throughput between old and new
func BenchmarkComparisonThroughput(b *testing.B) {
	bodies := []string{
		`{"keyId": "key_1", "roles": ["a"]}`,
		`{"keyId": "key_2", "roles": ["a", "b"]}`,
		`{"keyId": "key_3", "roles": ["a", "b", "c"]}`,
		`{"keyId": "key_4", "roles": ["a", "b", "c", "d"]}`,
		`{"keyId": "key_5", "roles": ["a", "b", "c", "d", "e"]}`,
	}

	b.Run("new/throughput_varied", func(b *testing.B) {
		v, err := New()
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				body := bodies[i%len(bodies)]
				req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer test_key_abc123")
				_, _ = v.Validate(context.Background(), req)
				i++
			}
		})
	})

	b.Run("old/throughput_varied", func(b *testing.B) {
		v, err := NewOldValidator()
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			i := 0
			for pb.Next() {
				body := bodies[i%len(bodies)]
				req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer test_key_abc123")
				_ = v.ValidateRequest(req)
				i++
			}
		})
	})
}

// BenchmarkComparisonLatency compares latency between old and new
func BenchmarkComparisonLatency(b *testing.B) {
	body := `{"keyId": "key_123abc", "roles": ["admin", "user"]}`

	b.Run("new/latency_sequential", func(b *testing.B) {
		v, err := New()
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test_key_abc123")
			_, _ = v.Validate(context.Background(), req)
		}
	})

	b.Run("old/latency_sequential", func(b *testing.B) {
		v, err := NewOldValidator()
		if err != nil {
			b.Fatal(err)
		}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test_key_abc123")
			_ = v.ValidateRequest(req)
		}
	})
}

// TestValidatorParity verifies that the new validator produces the same
// validation results as the old libopenapi validator. This ensures we don't
// accept invalid requests or reject valid ones compared to the reference.
func TestValidatorParity(t *testing.T) {
	newV, err := New()
	require.NoError(t, err)

	oldV, err := NewOldValidator()
	require.NoError(t, err)

	testCases := []struct {
		name        string
		method      string
		path        string
		body        string
		contentType string
		auth        string
		wantValid   bool // Expected result from both validators
	}{
		// Valid requests - both should accept
		{
			name:        "valid_simple_request",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_123abc", "roles": ["admin"]}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			wantValid:   true,
		},
		{
			name:        "valid_complex_request",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_123abc", "roles": ["admin:read", "admin:write", "user:*"]}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			wantValid:   true,
		},
		{
			name:        "valid_multiple_roles",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_123abc", "roles": ["admin", "user", "viewer"]}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			wantValid:   true,
		},
		{
			name:        "valid_empty_roles",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_123abc", "roles": []}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			wantValid:   true,
		},
		{
			name:        "valid_content_type_with_charset",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_123abc", "roles": ["admin"]}`,
			contentType: "application/json; charset=utf-8",
			auth:        "Bearer test_key",
			wantValid:   true,
		},

		// Invalid requests - both should reject
		{
			name:        "invalid_missing_keyId",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"roles": ["admin"]}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			wantValid:   false,
		},
		{
			name:        "invalid_missing_roles",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_123abc"}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			wantValid:   false,
		},
		{
			name:        "invalid_wrong_type_keyId",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": 123, "roles": ["admin"]}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			wantValid:   false,
		},
		{
			name:        "invalid_wrong_type_roles",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_123abc", "roles": "admin"}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			wantValid:   false,
		},
		{
			name:        "invalid_malformed_json",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_123abc", "roles": ["admin"`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			wantValid:   false,
		},
		{
			name:        "invalid_additional_properties",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_123abc", "roles": ["admin"], "extraField": "value"}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			wantValid:   false,
		},
		{
			name:        "invalid_null_keyId",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": null, "roles": ["admin"]}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			wantValid:   false,
		},
		{
			name:        "invalid_null_roles",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_123abc", "roles": null}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			wantValid:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			// Create requests for both validators
			newReq := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			newReq.Header.Set("Content-Type", tc.contentType)
			if tc.auth != "" {
				newReq.Header.Set("Authorization", tc.auth)
			}

			oldReq := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			oldReq.Header.Set("Content-Type", tc.contentType)
			if tc.auth != "" {
				oldReq.Header.Set("Authorization", tc.auth)
			}

			// Validate with both
			_, newValid := newV.Validate(context.Background(), newReq)
			oldValid := oldV.ValidateRequest(oldReq)

			// Log the result for visibility
			t.Logf("%s: old=%v, new=%v, want=%v", tc.name, oldValid, newValid, tc.wantValid)

			// The new validator should produce the same result as the old one
			// This ensures we're not more permissive than the reference implementation
			require.Equal(t, oldValid, newValid,
				"Validator parity failed for %s: old=%v, new=%v",
				tc.name, oldValid, newValid)

			// Also verify both match expected result
			require.Equal(t, tc.wantValid, newValid,
				"Expected %s to be valid=%v but got old=%v, new=%v",
				tc.name, tc.wantValid, oldValid, newValid)
		})
	}
}

// TestValidatorParityEdgeCases tests edge cases that might differ between implementations
func TestValidatorParityEdgeCases(t *testing.T) {
	newV, err := New()
	require.NoError(t, err)

	oldV, err := NewOldValidator()
	require.NoError(t, err)

	edgeCases := []struct {
		name        string
		method      string
		path        string
		body        string
		contentType string
		auth        string
		description string
	}{
		{
			name:        "unicode_in_keyId",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_ñoño_123", "roles": ["admin"]}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			description: "Unicode characters in string field",
		},
		{
			name:        "very_long_keyId",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_` + strings.Repeat("a", 1000) + `", "roles": ["admin"]}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			description: "Very long string value",
		},
		{
			name:        "many_roles",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_123", "roles": ["r1","r2","r3","r4","r5","r6","r7","r8","r9","r10"]}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			description: "Many items in array",
		},
		{
			name:        "empty_string_keyId",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "", "roles": ["admin"]}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			description: "Empty string for required field",
		},
		{
			name:        "whitespace_only_keyId",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "   ", "roles": ["admin"]}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			description: "Whitespace-only string",
		},
		{
			name:        "nested_empty_object_in_roles",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_123", "roles": [{}]}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			description: "Empty object in array",
		},
		{
			name:        "special_chars_in_strings",
			method:      "POST",
			path:        "/v2/keys.setRoles",
			body:        `{"keyId": "key_\"test\"_123", "roles": ["admin\nuser"]}`,
			contentType: "application/json",
			auth:        "Bearer test_key",
			description: "Special characters and escapes",
		},
	}

	for _, tc := range edgeCases {
		t.Run(tc.name, func(t *testing.T) {
			newReq := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			newReq.Header.Set("Content-Type", tc.contentType)
			if tc.auth != "" {
				newReq.Header.Set("Authorization", tc.auth)
			}

			oldReq := httptest.NewRequest(tc.method, tc.path, strings.NewReader(tc.body))
			oldReq.Header.Set("Content-Type", tc.contentType)
			if tc.auth != "" {
				oldReq.Header.Set("Authorization", tc.auth)
			}

			_, newValid := newV.Validate(context.Background(), newReq)
			oldValid := oldV.ValidateRequest(oldReq)

			// Log the result for edge cases (informational)
			t.Logf("%s: old=%v, new=%v (%s)", tc.name, oldValid, newValid, tc.description)

			// Verify parity
			require.Equal(t, oldValid, newValid,
				"Validator parity failed for edge case %s: old=%v, new=%v (%s)",
				tc.name, oldValid, newValid, tc.description)
		})
	}
}
