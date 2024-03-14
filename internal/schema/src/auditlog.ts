import { z } from "zod";

export const unkeyAuditLogEvents = z.enum([
  "workspace.create",
  "workspace.update",
  "workspace.delete",
  "workspace.opt_in",
  "api.create",
  "api.update",
  "api.delete",
  "key.create",
  "key.update",
  "key.delete",
  "ratelimitNamespace.create",
  "ratelimitNamespace.update",
  "ratelimitNamespace.delete",
  "ratelimitIdentifier.create",
  "ratelimitIdentifier.update",
  "ratelimitIdentifier.delete",
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
  time: z.number(),
  meta: z.record(z.union([z.string(), z.number(), z.boolean(), z.null()])).optional(),
  actor: z.object({
    type: z.string(),
    id: z.string(),
    name: z.string().optional(),
    meta: z.record(z.union([z.string(), z.number(), z.boolean(), z.null()])).optional(),
  }),
  resources: z.array(
    z.object({
      type: z.string(),
      id: z.string(),
      name: z.string().optional(),
      meta: z.record(z.union([z.string(), z.number(), z.boolean(), z.null()])).optional(),
    }),
  ),
  context: z.object({
    location: z.string(),
    userAgent: z.string().optional(),
  }),
});
