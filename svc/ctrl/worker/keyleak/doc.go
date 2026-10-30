// Package keyleak disables leaked keys and records leak audits in one transaction
// per batch before delivering alerts. Requests contain hashes, not raw tokens.
// NotifyBatch owns this durable write; Notify keeps its journal shape so pending
// alert invocations can replay across deployments.
// Slack and each recipient's email retry independently. Slack delivery is at
// least once; email also uses provider idempotency within its retention window.
package keyleak
