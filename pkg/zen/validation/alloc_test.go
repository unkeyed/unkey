//go:build benchmark

package validation

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http/httptest"
	"testing"
)

var allocTestValidator *Validator
var allocArrayValid = `[1,2,3,4,5,6,7,8,9,10,11,12,13,14,15,16,17,18,19,20,21,22,23,24,25,26,27,28,29,30,31,32,33,34,35,36,37,38,39,40,41,42,43,44,45,46,47,48,49,50]`

func initAllocTestValidator() {
	if allocTestValidator != nil {
		return
	}
	// Use the same spec as the main benchmark
	parser, err := NewSpecParser([]byte(benchmarkSpec))
	if err != nil {
		panic(fmt.Sprintf("failed to parse spec: %v", err))
	}
	compiler, err := NewSchemaCompiler(parser, []byte(benchmarkSpec))
	if err != nil {
		panic(fmt.Sprintf("failed to compile schemas: %v", err))
	}
	matcher := NewPathMatcher(parser.Operations())
	allocTestValidator = &Validator{
		matcher:         matcher,
		compiler:        compiler,
		securitySchemes: parser.SecuritySchemes(),
	}
}

// Test to understand where allocations come from in array validation
func BenchmarkArrayAllocation_Breakdown(b *testing.B) {
	initAllocTestValidator()

	// Benchmark just creating the request
	b.Run("1_CreateRequest", func(b *testing.B) {
		body := []byte(allocArrayValid)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/numbers", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test_token")
			_ = req
		}
	})

	// Benchmark JSON unmarshal only
	b.Run("2_JSONUnmarshal", func(b *testing.B) {
		body := []byte(allocArrayValid)
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			var data any
			json.Unmarshal(body, &data)
		}
	})

	// Benchmark schema validation only (no request overhead)
	b.Run("3_SchemaValidateOnly", func(b *testing.B) {
		// Get compiled schema via operationID
		compiledOp := allocTestValidator.compiler.GetOperation("postNumbers")
		if compiledOp == nil || compiledOp.BodySchema == nil {
			b.Fatal("no schema found")
		}

		// Pre-parse the JSON
		var data any
		json.Unmarshal([]byte(allocArrayValid), &data)

		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			compiledOp.BodySchema.Validate(data)
		}
	})

	// Benchmark full validation
	b.Run("4_FullValidation", func(b *testing.B) {
		b.ResetTimer()
		for i := 0; i < b.N; i++ {
			req := httptest.NewRequest("POST", "/numbers", bytes.NewReader([]byte(allocArrayValid)))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("Authorization", "Bearer test_token")
			allocTestValidator.Validate(context.Background(), req)
		}
	})
}

// Test with smaller arrays to see scaling
func BenchmarkArraySize_Scaling(b *testing.B) {
	initAllocTestValidator()
	compiledOp := allocTestValidator.compiler.GetOperation("postNumbers")
	if compiledOp == nil || compiledOp.BodySchema == nil {
		b.Fatal("no schema found")
	}

	sizes := []int{1, 5, 10, 25, 50}
	for _, size := range sizes {
		name := fmt.Sprintf("%02d_elements", size)
		b.Run(name, func(b *testing.B) {
			// Build array of given size
			arr := make([]any, size)
			for i := 0; i < size; i++ {
				arr[i] = float64(i + 1) // JSON numbers are float64
			}

			b.ResetTimer()
			for i := 0; i < b.N; i++ {
				compiledOp.BodySchema.Validate(arr)
			}
		})
	}
}
