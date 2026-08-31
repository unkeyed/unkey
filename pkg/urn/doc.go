// Package urn constructs and parses Unkey resource names.
//
// A v1 resource name has four colon-separated fields:
//
//	unkey:v1:{workspace_id}:{resource_path}
//
// A resource permission adds an action suffix:
//
//	unkey:v1:{workspace_id}:{resource_path}#{action}
//
// [ParseV1] accepts both concrete resource names and resource-name patterns:
// "*" matches one path segment and a trailing "/**" matches all descendants.
// [V1.Covers] decides whether a pattern covers a concrete name. Callers that
// require one concrete resource, such as audit logs, must reject wildcard
// paths themselves. Permission action types and authorization semantics live
// outside this package. Typed resources only bind actions that they accept.
//
// Resource paths follow the hierarchy in the resource permission catalog.
// [New] exposes typed resource builders for this hierarchy. For example:
//
//	urn.New().Workspace("ws_123").Project("proj_123").App("app_123").Any()
//
// See:
//   - docs/engineering/architecture/resources/unkey-resource-names.mdx
//   - docs/engineering/architecture/authorization/resource-permissions.mdx
//   - docs/engineering/architecture/authorization/resource-permission-catalog.mdx
package urn
