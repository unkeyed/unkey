// Command cliguidance extracts canonical CLI request metadata from OpenAPI.
package main

import (
	"encoding/json"
	"flag"
	"fmt"
	"go/ast"
	"go/parser"
	"go/token"
	"os"
	"slices"
	"strconv"
	"strings"
	"unicode"

	"gopkg.in/yaml.v3"
)

var endpointOverrides = map[string]string{
	"deployments create-deployment": "/v3/deployments.createDeployment",
}

func main() {
	spec := flag.String("spec", "svc/api/openapi/openapi-generated.yaml", "OpenAPI source")
	resources := flag.String("resources", "cmd/api/resources.go", "Go source declaring resourcePaths")
	out := flag.String("out", "cmd/api/internal/guidance/schemas_generated.json", "Output JSON")
	flag.Parse()
	if err := generate(*spec, *resources, *out); err != nil {
		fmt.Fprintln(os.Stderr, err)
		os.Exit(1)
	}
}

func generate(specPath, resourcesPath, outputPath string) error {
	spec, err := os.ReadFile(specPath)
	if err != nil {
		return fmt.Errorf("read OpenAPI: %w", err)
	}
	keys, err := resourceKeys(resourcesPath)
	if err != nil {
		return err
	}
	data, err := extract(spec, keys)
	if err != nil {
		return err
	}
	if err := os.WriteFile(outputPath, data, 0o644); err != nil {
		return fmt.Errorf("write schemas: %w", err)
	}
	return nil
}

func resourceKeys(path string) ([]string, error) {
	file, err := parser.ParseFile(token.NewFileSet(), path, nil, 0)
	if err != nil {
		return nil, fmt.Errorf("parse resources: %w", err)
	}
	for _, declaration := range file.Decls {
		general, ok := declaration.(*ast.GenDecl)
		if !ok || general.Tok != token.VAR {
			continue
		}
		for _, spec := range general.Specs {
			value, ok := spec.(*ast.ValueSpec)
			if !ok || len(value.Names) != 1 || value.Names[0].Name != "resourcePaths" || len(value.Values) != 1 {
				continue
			}
			literal, ok := value.Values[0].(*ast.CompositeLit)
			if !ok {
				return nil, fmt.Errorf("resourcePaths must be a map literal")
			}
			var keys []string
			for _, element := range literal.Elts {
				entry, ok := element.(*ast.KeyValueExpr)
				if !ok {
					return nil, fmt.Errorf("resourcePaths entry must have a key")
				}
				key, ok := entry.Key.(*ast.BasicLit)
				if !ok || key.Kind != token.STRING {
					return nil, fmt.Errorf("resourcePaths key must be a string literal")
				}
				name, err := strconv.Unquote(key.Value)
				if err != nil {
					return nil, fmt.Errorf("parse resource key: %w", err)
				}
				if slices.Contains(keys, name) {
					return nil, fmt.Errorf("duplicate resource key %q", name)
				}
				keys = append(keys, name)
			}
			if len(keys) == 0 {
				return nil, fmt.Errorf("resourcePaths is empty")
			}
			slices.Sort(keys)
			return keys, nil
		}
	}
	return nil, fmt.Errorf("resourcePaths not found in %s", path)
}

func extract(data []byte, keys []string) ([]byte, error) {
	var document map[string]any
	if err := yaml.Unmarshal(data, &document); err != nil {
		return nil, fmt.Errorf("parse OpenAPI: %w", err)
	}
	paths, ok := document["paths"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("OpenAPI paths not found")
	}
	schemas := make(map[string]any, len(keys))
	for path, raw := range paths {
		_, route, ok := strings.Cut(strings.TrimPrefix(path, "/"), "/")
		if !ok {
			continue
		}
		group, operation, ok := strings.Cut(route, ".")
		if !ok {
			continue
		}
		key := kebab(group) + " " + kebab(operation)
		if !slices.Contains(keys, key) {
			continue
		}
		selected, overridden := endpointOverrides[key]
		if !overridden {
			selected = "/v2/" + route
		}
		if path != selected {
			continue
		}
		if _, exists := schemas[key]; exists {
			return nil, fmt.Errorf("duplicate endpoint for %q", key)
		}
		item, ok := raw.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: invalid path item", path)
		}
		op, ok := item["post"].(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: POST operation not found", path)
		}
		schema, err := requestSchema(document, op)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}
		schemas[key] = schema
	}
	for _, key := range keys {
		if _, ok := schemas[key]; !ok {
			return nil, fmt.Errorf("canonical command %q has no OpenAPI endpoint", key)
		}
	}
	keys = slices.Clone(keys)
	slices.Sort(keys)
	encoded := []byte("{\n")
	for i, key := range keys {
		entry, err := json.Marshal(map[string]any{key: schemas[key]})
		if err != nil {
			return nil, fmt.Errorf("encode %s: %w", key, err)
		}
		encoded = append(encoded, '\t')
		encoded = append(encoded, entry[1:len(entry)-1]...)
		if i < len(keys)-1 {
			encoded = append(encoded, ',')
		}
		encoded = append(encoded, '\n')
	}
	return append(encoded, '}', '\n'), nil
}

func requestSchema(document, operation map[string]any) (map[string]any, error) {
	raw, exists := operation["requestBody"]
	if !exists {
		return map[string]any{"type": "object"}, nil
	}
	body, ok := raw.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid request body")
	}
	if ref, ok := body["$ref"].(string); ok {
		resolved, err := resolve(document, ref)
		if err != nil {
			return nil, err
		}
		body, ok = resolved.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("%s: invalid request body", ref)
		}
	}
	content, ok := body["content"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("request body has no content")
	}
	media, ok := content["application/json"].(map[string]any)
	if !ok {
		return nil, fmt.Errorf("request body has no application/json content")
	}
	raw, exists = media["schema"]
	if !exists {
		return nil, fmt.Errorf("request body has no schema")
	}
	return compact(document, raw, make(map[string]bool))
}

func resolve(document map[string]any, ref string) (any, error) {
	if !strings.HasPrefix(ref, "#/") {
		return nil, fmt.Errorf("external reference %q is unsupported", ref)
	}
	var current any = document
	for _, token := range strings.Split(strings.TrimPrefix(ref, "#/"), "/") {
		object, ok := current.(map[string]any)
		if !ok {
			return nil, fmt.Errorf("invalid reference %q", ref)
		}
		key := strings.ReplaceAll(strings.ReplaceAll(token, "~1", "/"), "~0", "~")
		current, ok = object[key]
		if !ok {
			return nil, fmt.Errorf("missing reference %q", ref)
		}
	}
	return current, nil
}

func kebab(value string) string {
	runes := []rune(value)
	var out strings.Builder
	for i, r := range runes {
		if unicode.IsUpper(r) && i > 0 && (unicode.IsLower(runes[i-1]) || unicode.IsDigit(runes[i-1]) || (i+1 < len(runes) && unicode.IsLower(runes[i+1]))) {
			out.WriteByte('-')
		}
		out.WriteRune(unicode.ToLower(r))
	}
	return out.String()
}
