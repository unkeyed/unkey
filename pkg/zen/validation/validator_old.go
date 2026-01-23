//go:build benchmark_comparison

package validation

// This file contains the old libopenapi-based validator implementation.
// It is excluded from normal builds (requires -tags=benchmark_comparison).
//
// To run comparison benchmarks:
//   bazel run //pkg/zen/validation:validation_test -- -test.bench=BenchmarkComparison -test.benchmem -tags=benchmark_comparison
//
// Or with go test directly:
//   go test -bench=BenchmarkComparison -benchmem -tags=benchmark_comparison

import (
	"bytes"
	"io"
	"net/http"

	"github.com/pb33f/libopenapi"
	validator "github.com/pb33f/libopenapi-validator"
	"github.com/unkeyed/unkey/svc/api/openapi"
)

// OldValidator is the original libopenapi-based validator for comparison benchmarks
type OldValidator struct {
	validator validator.Validator
}

// NewOldValidator creates a new OldValidator using libopenapi
func NewOldValidator() (*OldValidator, error) {
	document, err := libopenapi.NewDocument(openapi.Spec)
	if err != nil {
		return nil, err
	}

	v, errs := validator.NewValidator(document)
	if len(errs) > 0 {
		return nil, errs[0]
	}

	return &OldValidator{
		validator: v,
	}, nil
}

// ValidateRequest validates an HTTP request using the old libopenapi validator
// Returns true if valid, false otherwise
func (v *OldValidator) ValidateRequest(r *http.Request) bool {
	// Read the body
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		return false
	}
	r.Body = io.NopCloser(bytes.NewReader(bodyBytes))

	valid, _ := v.validator.ValidateHttpRequest(r)
	return valid
}
