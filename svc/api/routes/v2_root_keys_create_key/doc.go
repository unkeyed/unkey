// Package handler creates system root keys for authorized principals.
// Keys, grants, and customer audit events commit together. Legacy inputs also
// store their URN equivalents; URN inputs do not generate legacy grants.
// Resource resolution uses the customer workspace; storage uses the configured
// internal workspace, keyspace, and project, without an identity association.
//
// Creation requires rootKeys/*#write, including canonical global grants.
// Creation and delegation cannot exceed the caller's permissions.
// The translation allowlist excludes API create/update and aggregate reads:
// generic URN actions can combine legacy capabilities. Mappings require proof
// of scope and action equivalence in both directions.
package handler
