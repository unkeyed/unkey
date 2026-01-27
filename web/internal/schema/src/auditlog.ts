import { z } from "zod";

export const unkeyAuditLogEvents = z.enum([
  "workspace.create",
  "workspace.update",
  "workspace.delete",
  "workspace.opt_in",
  "sentinel.create",
  "llmSentinel.create",
  "llmSentinel.delete",
  "api.create",
  "api.update",
  "api.delete",
  "key.create",
  "key.reroll",
  "key.update",
  "key.delete",
  "ratelimitNamespace.create",
  "ratelimitNamespace.update",
  "ratelimitNamespace.delete",
  "vercelIntegration.create",
  "vercelIntegration.update",
  "vercelIntegration.delete",
  "vercelBinding.create",
  "vercelBinding.update",
  "vercelBinding.delete",
  "role.create",
  "role.update",
  "role.delete",
  "permission.create",
  "permission.update",
  "permission.delete",
  "authorization.connect_role_and_permission",
  "authorization.disconnect_role_and_permissions",
  "authorization.connect_role_and_key",
  "authorization.disconnect_role_and_key",
  "authorization.connect_permission_and_key",
  "authorization.disconnect_permission_and_key",
  "secret.create",
  "secret.decrypt",
  "secret.update",
  "webhook.create",
  "webhook.update",
  "webhook.delete",
  "reporter.create",
  "identity.create",
  "identity.update",
  "identity.delete",
  "ratelimit.create",
  "ratelimit.update",
  "ratelimit.delete",
  "ratelimit.set_override",
  "ratelimit.read_override",
  "ratelimit.delete_override",
  "auditLogBucket.create",
  "project.create",
  "environment.create",
  "deployment.rollback",
  "deployment.promote",
]);

export const auditLogSchemaV1 = z.object({
  /**
   * The workspace owning this audit log
   */
  workspaceId: z.string(),

  /**
   * Buckets are used as namespaces for different logs belonging to a single workspace
   */
  bucket: z.string(),
  auditLogId: z.string(),
  event: z.string(),
  description: z.string().optional(),
  time: z.number().prefault(() => Date.now()),
  meta: z
    .record(z.string(), z.union([z.string(), z.number(), z.boolean(), z.null(), z.undefined()]))
    .optional(),
  actor: z.object({
    type: z.string(),
    id: z.string(),
    name: z.string().optional(),
    meta: z
      .record(z.string(), z.union([z.string(), z.number(), z.boolean(), z.null(), z.undefined()]))
      .optional(),
  }),
  resources: z.array(
    z.object({
      type: z.string(),
      id: z.string(),
      name: z.string().optional(),
      meta: z
        .record(z.string(), z.union([z.string(), z.number(), z.boolean(), z.null(), z.undefined()]))
        .optional(),
    }),
  ),
  context: z.object({
    location: z.string(),
    userAgent: z.string().optional(),
  }),
});
