package validation

import (
	"bytes"
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

// BenchmarkValidatorInit benchmarks the initialization of the validator
func BenchmarkValidatorInit(b *testing.B) {
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := New()
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkValidateSingleRequest benchmarks validating a single request
func BenchmarkValidateSingleRequest(b *testing.B) {
	v, err := New()
	if err != nil {
		b.Fatal(err)
	}

	body := `{"keyId": "key_123abc", "roles": ["admin", "user"]}`

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test_key_abc123")
		_, _ = v.Validate(context.Background(), req)
	}
}

// BenchmarkValidateParallel benchmarks parallel request validation
func BenchmarkValidateParallel(b *testing.B) {
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

// BenchmarkSchemaValidation benchmarks validation with different body sizes
func BenchmarkSchemaValidation(b *testing.B) {
	v, err := New()
	if err != nil {
		b.Fatal(err)
	}

	b.Run("SmallBody_100B", func(b *testing.B) {
		// ~100 bytes body
		body := `{"keyId": "key_123abc", "roles": ["admin"]}`
		benchmarkWithBody(b, v, body)
	})

	b.Run("MediumBody_1KB", func(b *testing.B) {
		// ~1KB body with more roles
		roles := make([]string, 20)
		for i := range roles {
			roles[i] = "role_" + strings.Repeat("x", 40)
		}
		body := `{"keyId": "key_123abc", "roles": ["` + strings.Join(roles, `","`) + `"]}`
		benchmarkWithBody(b, v, body)
	})

	b.Run("LargeBody_10KB", func(b *testing.B) {
		// ~10KB body with many roles
		roles := make([]string, 200)
		for i := range roles {
			roles[i] = "role_" + strings.Repeat("x", 40)
		}
		body := `{"keyId": "key_123abc", "roles": ["` + strings.Join(roles, `","`) + `"]}`
		benchmarkWithBody(b, v, body)
	})
}

func benchmarkWithBody(b *testing.B, v *Validator, body string) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test_key_abc123")
		_, _ = v.Validate(context.Background(), req)
	}
}

// BenchmarkPathMatching benchmarks path matching performance
func BenchmarkPathMatching(b *testing.B) {
	ops := map[string]*Operation{
		"POST /v2/keys.setRoles": {
			Method:      "POST",
			Path:        "/v2/keys.setRoles",
			OperationID: "keys.setRoles",
		},
		"GET /v2/keys.getKey": {
			Method:      "GET",
			Path:        "/v2/keys.getKey",
			OperationID: "keys.getKey",
		},
		"POST /users/{userId}": {
			Method:      "POST",
			Path:        "/users/{userId}",
			OperationID: "users.update",
		},
		"GET /users/{userId}/posts/{postId}": {
			Method:      "GET",
			Path:        "/users/{userId}/posts/{postId}",
			OperationID: "users.posts.get",
		},
	}

	matcher := NewPathMatcher(ops)

	b.Run("ExactMatch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = matcher.Match("POST", "/v2/keys.setRoles")
		}
	})

	b.Run("TemplateMatch_SingleParam", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = matcher.Match("POST", "/users/user_123abc")
		}
	})

	b.Run("TemplateMatch_MultiParam", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = matcher.Match("GET", "/users/user_123abc/posts/post_456def")
		}
	})

	b.Run("NoMatch", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_, _ = matcher.Match("DELETE", "/unknown/path")
		}
	})
}

// BenchmarkSecurityValidation benchmarks security validation
func BenchmarkSecurityValidation(b *testing.B) {
	schemes := map[string]SecurityScheme{
		"bearerAuth": {
			Type:   SecurityTypeHTTP,
			Scheme: "bearer",
		},
	}

	requirements := []SecurityRequirement{
		{Schemes: map[string][]string{"bearerAuth": {}}},
	}

	b.Run("ValidBearer", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			req.Header.Set("Authorization", "Bearer valid_token_abc123")
			_ = ValidateSecurity(req, requirements, schemes, "req-123")
		}
	})

	b.Run("MissingAuth", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("GET", "/test", nil)
			_ = ValidateSecurity(req, requirements, schemes, "req-123")
		}
	})
}

// BenchmarkParameterParsing benchmarks parameter style parsing
func BenchmarkParameterParsing(b *testing.B) {
	b.Run("FormStyle_Simple", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = ParseByStyle("form", true, []string{"value1", "value2", "value3"}, "array", nil, "param")
		}
	})

	b.Run("FormStyle_CommaSeparated", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = ParseByStyle("form", false, []string{"value1,value2,value3"}, "array", nil, "param")
		}
	})

	b.Run("SimpleStyle_Array", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = ParseByStyle("simple", false, []string{"value1,value2,value3"}, "array", nil, "param")
		}
	})

	b.Run("PipeDelimited", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			_ = ParseByStyle("pipeDelimited", false, []string{"value1|value2|value3"}, "array", nil, "param")
		}
	})
}

// BenchmarkErrorTransform benchmarks error transformation
func BenchmarkErrorTransform(b *testing.B) {
	v, err := New()
	if err != nil {
		b.Fatal(err)
	}

	// Create an invalid request to generate errors
	invalidBody := `{"roles": ["admin"]}` // Missing required keyId

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v2/keys.setRoles", bytes.NewReader([]byte(invalidBody)))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test_key_abc123")
		_, _ = v.Validate(context.Background(), req)
	}
}

// BenchmarkContentTypeValidation benchmarks content type validation
func BenchmarkContentTypeValidation(b *testing.B) {
	v, err := New()
	if err != nil {
		b.Fatal(err)
	}

	body := `{"keyId": "key_123abc", "roles": ["admin"]}`

	b.Run("ValidContentType", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test_key_abc123")
			_, _ = v.Validate(context.Background(), req)
		}
	})

	b.Run("ContentTypeWithCharset", func(b *testing.B) {
		b.ReportAllocs()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json; charset=utf-8")
			req.Header.Set("Authorization", "Bearer test_key_abc123")
			_, _ = v.Validate(context.Background(), req)
		}
	})
}

// BenchmarkFullValidationPipeline benchmarks the complete validation pipeline
func BenchmarkFullValidationPipeline(b *testing.B) {
	v, err := New()
	if err != nil {
		b.Fatal(err)
	}

	testCases := []struct {
		name string
		body string
	}{
		{
			name: "MinimalValid",
			body: `{"keyId": "k", "roles": []}`,
		},
		{
			name: "TypicalValid",
			body: `{"keyId": "key_123abc", "roles": ["admin", "user", "viewer"]}`,
		},
		{
			name: "ComplexValid",
			body: `{"keyId": "key_123abc", "roles": [{"roleId": "role_1"}, {"roleId": "role_2"}, {"roleId": "role_3"}]}`,
		},
	}

	for _, tc := range testCases {
		b.Run(tc.name, func(b *testing.B) {
			b.ReportAllocs()
			for i := 0; i < b.N; i++ {
				req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
				req.Header.Set("Authorization", "Bearer test_key_abc123")
				_, _ = v.Validate(context.Background(), req)
			}
		})
	}
}
