import { InferModel } from "drizzle-orm";
import * as schema from "./schema";

export type Key = InferModel<typeof schema.keys>;
export type Role = InferModel<typeof schema.roles>;
export type Api = InferModel<typeof schema.apis>;
export type Workspace = InferModel<typeof schema.workspaces>;
export type KeyAuth = InferModel<typeof schema.keyAuth>;
export type VercelIntegration = InferModel<typeof schema.vercelIntegrations>;
export type VercelBinding = InferModel<typeof schema.vercelBindings>;
export type AuditLog = InferModel<typeof schema.auditLogs>;
