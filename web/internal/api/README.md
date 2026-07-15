# Dashboard API SDK

This private workspace package is generated from
`svc/api/openapi/openapi-generated.yaml` using the public SDK's Speakeasy
configuration.

Run `mise run generate` from the repository root after changing the API
specification. It invokes `generate-api-sdk`; commit the generated `src` changes
with the specification change.

The generated source is committed, while `esm` is built locally or during the
dashboard build and remains ignored. This ensures Next.js and Vercel consume real
ESM files rather than resolving generated `.js` imports against TypeScript source.
