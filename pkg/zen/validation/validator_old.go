//go:build ignore

package validation

// This file contains the old libopenapi-based validator implementation.
// It is IGNORED by default and kept only as documentation/reference.
//
// To run comparison benchmarks, you must:
// 1. Change the build tag above from "ignore" to "benchmark_comparison"
// 2. Add libopenapi to go.mod:
//      go get github.com/pb33f/libopenapi@v0.21.12
//      go get github.com/pb33f/libopenapi-validator@v0.4.2
// 3. Uncomment the validation_comparison_test target in BUILD.bazel
// 4. Run: bazel run //pkg/zen/validation:validation_comparison_test -- -test.bench=. -test.benchmem

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
