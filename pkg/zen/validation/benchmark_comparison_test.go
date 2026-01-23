//go:build !benchmark_comparison

package validation

import (
	"context"
	"net/http/httptest"
	"strings"
	"testing"
)

// This file contains benchmarks for the NEW validator only.
// To run comparison benchmarks against the old libopenapi implementation,
// use the benchmark_comparison build tag:
//
//   go test -bench=. -benchmem -tags=benchmark_comparison
//
// See benchmark_comparison_old_test.go for the comparison benchmarks.

// BenchmarkComparison provides benchmarks for the new validator implementation.
//
// Run these benchmarks:
//
//	bazel run //pkg/zen/validation:validation_test -- -test.bench=BenchmarkComparison -test.benchmem
//
// Expected output format:
//
//	BenchmarkComparison/new/init-10           1000    1234567 ns/op    123456 B/op    1234 allocs/op
//	BenchmarkComparison/new/validate-10      50000      23456 ns/op      1234 B/op      12 allocs/op
func BenchmarkComparison(b *testing.B) {
	b.Run("new/init", benchNewInit)
	b.Run("new/validate_simple", benchNewValidateSimple)
	b.Run("new/validate_complex", benchNewValidateComplex)
	b.Run("new/parallel", benchNewParallel)
	b.Run("new/invalid_request", benchNewInvalidRequest)
	b.Run("new/missing_auth", benchNewMissingAuth)
}

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

	// More complex body with nested objects
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

func benchNewInvalidRequest(b *testing.B) {
	v, err := New()
	if err != nil {
		b.Fatal(err)
	}

	// Invalid request - missing required field
	body := `{"roles": ["admin"]}`

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Authorization", "Bearer test_key_abc123")
		_, _ = v.Validate(context.Background(), req)
	}
}

func benchNewMissingAuth(b *testing.B) {
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
		// No Authorization header
		_, _ = v.Validate(context.Background(), req)
	}
}

// BenchmarkComparisonMemory provides memory-focused benchmarks
func BenchmarkComparisonMemory(b *testing.B) {
	b.Run("new/memory_per_request", func(b *testing.B) {
		v, err := New()
		if err != nil {
			b.Fatal(err)
		}

		body := `{"keyId": "key_123abc", "roles": ["admin", "user", "viewer"]}`

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test_key_abc123")
			_, _ = v.Validate(context.Background(), req)
		}
	})
}

// BenchmarkComparisonThroughput provides throughput-focused benchmarks
func BenchmarkComparisonThroughput(b *testing.B) {
	v, err := New()
	if err != nil {
		b.Fatal(err)
	}

	bodies := []string{
		`{"keyId": "key_1", "roles": ["a"]}`,
		`{"keyId": "key_2", "roles": ["a", "b"]}`,
		`{"keyId": "key_3", "roles": ["a", "b", "c"]}`,
		`{"keyId": "key_4", "roles": ["a", "b", "c", "d"]}`,
		`{"keyId": "key_5", "roles": ["a", "b", "c", "d", "e"]}`,
	}

	b.Run("new/throughput_varied", func(b *testing.B) {
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
}

// BenchmarkComparisonLatency provides latency-focused benchmarks
func BenchmarkComparisonLatency(b *testing.B) {
	v, err := New()
	if err != nil {
		b.Fatal(err)
	}

	body := `{"keyId": "key_123abc", "roles": ["admin", "user"]}`

	b.Run("new/latency_sequential", func(b *testing.B) {
		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/v2/keys.setRoles", strings.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test_key_abc123")
			_, _ = v.Validate(context.Background(), req)
		}
	})
}
