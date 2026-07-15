// Command upsert-workos-permissions syncs Unkey's WorkOS permission slugs into
// a WorkOS environment.
//
// Usage:
//
//	WORKOS_API_KEY=sk_... go run ./tools/upsert-workos-permissions
//	go run ./tools/upsert-workos-permissions -dry-run
package main

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/http"
	"net/url"
	"os"
	"slices"
	"strings"

	"github.com/unkeyed/unkey/pkg/auth/workos"
)

const defaultWorkOSAPIBaseURL = "https://api.workos.com"

var (
	workOSAPIBaseURL = defaultWorkOSAPIBaseURL
	workOSHTTPClient = http.DefaultClient
)

type config struct {
	apiKey string
	dryRun bool
}

type permissionBody struct {
	Slug        string `json:"slug,omitempty"`
	Name        string `json:"name,omitempty"`
	Description string `json:"description,omitempty"`
}

type permissionListResponse struct {
	Data         []permissionBody `json:"data"`
	ListMetadata struct {
		After *string `json:"after"`
	} `json:"list_metadata"`
}

type apiError struct {
	statusCode int
	body       string
}

func (e apiError) Error() string {
	return fmt.Sprintf("workos returned status %d: %s", e.statusCode, e.body)
}

func main() {
	cfg := config{
		apiKey: os.Getenv("WORKOS_API_KEY"),
		dryRun: false,
	}

	flag.StringVar(&cfg.apiKey, "api-key", cfg.apiKey, "WorkOS API key. Defaults to WORKOS_API_KEY.")
	flag.BoolVar(&cfg.dryRun, "dry-run", false, "Print planned changes without calling WorkOS.")
	flag.Parse()

	if err := run(context.Background(), cfg, workos.PermissionDefinitions(), os.Stdout); err != nil {
		_, _ = fmt.Fprintf(os.Stderr, "upsert workos permissions: %v\n", err)
		os.Exit(1)
	}
}

func run(ctx context.Context, cfg config, definitions []workos.PermissionDefinition, out io.Writer) error {
	if !cfg.dryRun && strings.TrimSpace(cfg.apiKey) == "" {
		return errors.New("api key is required unless -dry-run is set")
	}

	baseURL, err := url.Parse(workOSAPIBaseURL)
	if err != nil {
		return fmt.Errorf("parse api base URL: %w", err)
	}

	existingPermissions := []string{}
	if !cfg.dryRun {
		var err error
		existingPermissions, err = listExistingPermissions(ctx, workOSHTTPClient, *baseURL, cfg.apiKey)
		if err != nil {
			return fmt.Errorf("list existing permissions: %w", err)
		}
	}

	expectedPermissions := make(map[string]struct{}, len(definitions))
	for _, definition := range definitions {
		expectedPermissions[definition.Slug] = struct{}{}

		body := permissionBody{
			Slug:        definition.Slug,
			Name:        definition.Name,
			Description: definition.Description,
		}

		if cfg.dryRun {
			if _, err := fmt.Fprintf(out, "would upsert %s (%s)\n", body.Slug, body.Name); err != nil {
				return fmt.Errorf("write dry-run output: %w", err)
			}
			continue
		}

		action, err := upsertPermission(ctx, workOSHTTPClient, *baseURL, cfg.apiKey, body)
		if err != nil {
			return fmt.Errorf("%s: %w", definition.Slug, err)
		}
		if _, err := fmt.Fprintf(out, "%s %s\n", action, definition.Slug); err != nil {
			return fmt.Errorf("write output: %w", err)
		}
	}

	if !cfg.dryRun {
		if err := logUnmanagedPermissions(out, existingPermissions, expectedPermissions); err != nil {
			return err
		}
	}

	return nil
}

func upsertPermission(ctx context.Context, client *http.Client, baseURL url.URL, apiKey string, body permissionBody) (string, error) {
	_, err := request(ctx, client, baseURL, apiKey, http.MethodGet, "/authorization/permissions/"+body.Slug, nil, nil)
	if err == nil {
		updateBody := permissionBody{
			Slug:        "",
			Name:        body.Name,
			Description: body.Description,
		}
		_, updateErr := request(ctx, client, baseURL, apiKey, http.MethodPatch, "/authorization/permissions/"+body.Slug, nil, updateBody)
		if updateErr != nil {
			return "", updateErr
		}
		return "updated", nil
	}

	var apiErr apiError
	if !errors.As(err, &apiErr) || apiErr.statusCode != http.StatusNotFound {
		return "", err
	}

	_, createErr := request(ctx, client, baseURL, apiKey, http.MethodPost, "/authorization/permissions", nil, body)
	if createErr != nil {
		return "", createErr
	}
	return "created", nil
}

func listExistingPermissions(ctx context.Context, client *http.Client, baseURL url.URL, apiKey string) ([]string, error) {
	slugs := []string{}
	after := ""

	for {
		query := url.Values{}
		if after != "" {
			query.Set("after", after)
		}

		data, err := request(ctx, client, baseURL, apiKey, http.MethodGet, "/authorization/permissions", query, nil)
		if err != nil {
			return nil, err
		}

		var response permissionListResponse
		if err := json.Unmarshal(data, &response); err != nil {
			return nil, fmt.Errorf("decode permission list: %w", err)
		}

		for _, permission := range response.Data {
			if permission.Slug != "" {
				slugs = append(slugs, permission.Slug)
			}
		}

		if response.ListMetadata.After == nil || *response.ListMetadata.After == "" {
			break
		}
		after = *response.ListMetadata.After
	}

	slices.Sort(slugs)
	return slugs, nil
}

func logUnmanagedPermissions(out io.Writer, existingPermissions []string, expectedPermissions map[string]struct{}) error {
	for _, slug := range existingPermissions {
		if _, ok := expectedPermissions[slug]; ok {
			continue
		}
		if _, err := fmt.Fprintf(out, "unmanaged %s\n", slug); err != nil {
			return fmt.Errorf("write unmanaged permission output: %w", err)
		}
	}

	return nil
}

func request(ctx context.Context, client *http.Client, baseURL url.URL, apiKey string, method string, path string, query url.Values, body any) ([]byte, error) {
	endpoint := baseURL
	endpoint.Path = strings.TrimRight(endpoint.Path, "/") + path
	endpoint.RawPath = strings.TrimRight(endpoint.RawPath, "/") + escapePath(path)
	endpoint.RawQuery = query.Encode()

	var reqBody io.Reader
	if body != nil {
		data, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("marshal request body: %w", err)
		}
		reqBody = bytes.NewReader(data)
	}

	req, err := http.NewRequestWithContext(ctx, method, endpoint.String(), reqBody)
	if err != nil {
		return nil, fmt.Errorf("build request: %w", err)
	}
	req.Header.Set("Authorization", "Bearer "+apiKey)
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("Accept", "application/json")

	res, err := client.Do(req)
	if err != nil {
		return nil, fmt.Errorf("send request: %w", err)
	}
	defer func() { _ = res.Body.Close() }()

	data, err := io.ReadAll(res.Body)
	if err != nil {
		return nil, fmt.Errorf("read response body: %w", err)
	}

	if res.StatusCode < 200 || res.StatusCode >= 300 {
		return nil, apiError{statusCode: res.StatusCode, body: strings.TrimSpace(string(data))}
	}

	return data, nil
}

func escapePath(path string) string {
	segments := strings.Split(path, "/")
	for i, segment := range segments {
		segments[i] = url.PathEscape(segment)
	}
	return strings.Join(segments, "/")
}
