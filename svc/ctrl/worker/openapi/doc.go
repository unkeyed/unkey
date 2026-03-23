// Package openapi provides a Restate virtual object that scrapes OpenAPI
// specifications from running user deployments.
//
// After a deployment completes successfully, the deploy workflow fires a
// one-way ScrapeSpec call to this service. The handler reads the spec path
// from the app's runtime settings (openapi_spec_path), resolves the target
// URL from the deployment's frontline routes, fetches the spec, and persists
// the response body into the openapi_specs table via [db.Query.UpsertOpenApiSpec].
//
// Absence is not an error: the handler returns success when scraping is not
// configured, when no suitable route or instance exists, or when the endpoint
// returns 404.
package openapi
