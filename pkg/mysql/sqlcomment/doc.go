// Package sqlcomment appends SQLCommenter-compatible metadata to SQL statements
// so PlanetScale Query Insights can attribute query load to services, operations,
// and deploys.
//
// Tags are injected at the database client boundary ([pkg/mysql.Replica]) so
// sqlc-generated call sites stay unchanged. sqlc operation names are parsed from
// the leading `-- name: OperationName` header that sqlc embeds in query strings.
//
// Dynamic tags (route, source) travel on [context.Context] via [WithDynamic].
// Static tags (application, service, region, release_sha) are set once per
// process when opening database connections. release_sha is link-time metadata
// from [github.com/unkeyed/unkey/pkg/buildinfo.Revision].
package sqlcomment
