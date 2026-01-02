import type { InferInsertModel, InferSelectModel } from "drizzle-orm";
import type * as schema from "./schema";

export type Key = InferSelectModel<typeof schema.keys>;
export type InsertKey = InferInsertModel<typeof schema.keys>;

export type Api = InferSelectModel<typeof schema.apis>;
export type InsertApi = InferInsertModel<typeof schema.apis>;

export type Workspace = InferSelectModel<typeof schema.workspaces>;
export type InsertWorkspace = InferInsertModel<typeof schema.workspaces>;

export type KeyAuth = InferSelectModel<typeof schema.keyAuth>;
export type InsertKeyAuth = InferInsertModel<typeof schema.keyAuth>;

export type VercelIntegration = InferSelectModel<typeof schema.vercelIntegrations>;
export type InsertVercelIntegration = InferInsertModel<typeof schema.vercelIntegrations>;

export type VercelBinding = InferSelectModel<typeof schema.vercelBindings>;
export type InsertVercelBinding = InferInsertModel<typeof schema.vercelBindings>;

export type Permission = InferSelectModel<typeof schema.permissions>;
export type InsertPermission = InferInsertModel<typeof schema.permissions>;

export type Role = InferSelectModel<typeof schema.roles>;
export type InsertRole = InferInsertModel<typeof schema.roles>;

export type RatelimitOverride = InferSelectModel<typeof schema.ratelimitOverrides>;
export type InsertRatelimitOverride = InferInsertModel<typeof schema.ratelimitOverrides>;

export type RatelimitNamespace = InferSelectModel<typeof schema.ratelimitNamespaces>;
export type InsertRatelimitNamespace = InferInsertModel<typeof schema.ratelimitNamespaces>;

export type EncryptedKey = InferSelectModel<typeof schema.encryptedKeys>;
export type InsertEncryptedKey = InferInsertModel<typeof schema.encryptedKeys>;

export type KeyRole = InferSelectModel<typeof schema.keysRoles>;
export type InsertKeyRole = InferInsertModel<typeof schema.keysRoles>;

export type KeyPermission = InferSelectModel<typeof schema.keysPermissions>;
export type InsertKeyPermission = InferInsertModel<typeof schema.keysPermissions>;

export type Ratelimit = InferSelectModel<typeof schema.ratelimits>;
export type InsertRatelimit = InferInsertModel<typeof schema.ratelimits>;

export type Identity = InferSelectModel<typeof schema.identities>;
export type InsertIdentity = InferInsertModel<typeof schema.identities>;

export type AuditLog = InferSelectModel<typeof schema.auditLog>;
export type InsertAuditLog = InferInsertModel<typeof schema.auditLog>;

export type AuditLogTarget = InferSelectModel<typeof schema.auditLogTarget>;
export type InsertAuditLogTarget = InferInsertModel<typeof schema.auditLogTarget>;

export type Quotas = InferSelectModel<typeof schema.quotas>;
export type InsertQuotas = InferInsertModel<typeof schema.quotas>;
