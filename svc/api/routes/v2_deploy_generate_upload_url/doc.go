// Package handler implements the POST /v2/deploy.generateUploadUrl endpoint
// for generating pre-signed S3 URLs used to upload deployment build contexts.
//
// This endpoint is part of the deployment workflow where clients upload their
// build artifacts to S3 before triggering a deployment. The handler delegates
// URL generation to the control plane service via gRPC, ensuring all upload
// URLs are centrally managed and consistently configured.
//
// # Authentication and Authorization
//
// Requests must include a valid root key in the Authorization header. The root
// key must have either wildcard project permission (project.*.generate_upload_url)
// or specific permission for the target project (project.<id>.generate_upload_url).
//
// The handler also verifies that the requested project belongs to the workspace
// associated with the root key. Requests for projects in other workspaces return
// 404 to avoid leaking information about project existence.
//
// # Request Flow
//
// The handler validates the root key, binds and validates the request body,
// checks RBAC permissions, verifies project ownership, then calls the control
// plane to generate the upload URL. On success, it returns both the pre-signed
// upload URL and the build context path where the uploaded artifact will be stored.
//
// # Error Responses
//
// The handler returns 400 for missing or invalid request body, 401 for invalid
// root keys, 403 for insufficient permissions, and 404 when the project does
// not exist or belongs to a different workspace.
package handler
