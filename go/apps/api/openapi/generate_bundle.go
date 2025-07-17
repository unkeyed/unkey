//go:build ignore

package main

import (
	"flag"
	"fmt"
	"log"
	"os"

	"github.com/pb33f/libopenapi"
	"github.com/pb33f/libopenapi/bundler"
	"github.com/pb33f/libopenapi/datamodel"
)

func main() {
	var (
		input  = flag.String("input", "openapi-split.yaml", "Input OpenAPI file")
		output = flag.String("output", "openapi-bundled.yaml", "Output bundled OpenAPI file")
	)
	flag.Parse()

	bytes, err := os.ReadFile(*input)
	if err != nil {
		log.Fatalf("Failed to read input file %s: %v", *input, err)
	}

	// Create OpenAPI config
	config := datamodel.NewDocumentConfiguration()
	config.BasePath = "."
	config.ExtractRefsSequentially = true
	// config.BundleInlineRefs = true

	// Parse the document
	document, err := libopenapi.NewDocumentWithConfiguration(bytes, config)
	if err != nil {
		log.Fatalf("Failed to parse OpenAPI document: %v", err)
	}

	// Build the v3 model
	v3Model, errs := document.BuildV3Model()
	if len(errs) > 0 {
		for _, e := range errs {
			log.Printf("Error building model: %v", e)
		}
		log.Fatal("Failed to build v3 model")
	}

	// Create bundle composition config
	bundleConfig := &bundler.BundleCompositionConfig{}

	// Bundle the document (this will compose references into components section)
	bundledBytes, err := bundler.BundleDocumentComposed(&v3Model.Model, bundleConfig)
	if err != nil {
		log.Fatalf("Failed to bundle document: %v", err)
	}

	// Write the bundled output
	err = os.WriteFile(*output, bundledBytes, 0644)
	if err != nil {
		log.Fatalf("Failed to write output file %s: %v", *output, err)
	}

	fmt.Printf("✅ Successfully bundled OpenAPI spec to %s\n", *output)
}
