// Package openapi provides a Restate virtual object that scrapes OpenAPI
// specifications from running user deployments.
//
// After a deployment completes successfully, the deploy workflow fires a
// one-way ScrapeSpec call to this service. The handler fetches /openapi.yaml
// from the deployment via its deployment-specific public FQDN, and persists
// the response body into the openapi_specs table via [db.Query.UpsertOpenApiSpec].
//
// If the endpoint returns 404 or a non-200 status, the handler logs the
// outcome and returns success — not all user deployments expose an OpenAPI
// spec, so absence is not an error.
package openapi
