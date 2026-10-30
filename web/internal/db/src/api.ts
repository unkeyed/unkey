import { sql } from "drizzle-orm";
import { apis } from "./schema/apis";
import type { InsertApi } from "./types";

// Drizzle inserts every schema column, including retired columns omitted from values.
// Keep this projection explicit so schema-first column removal is safe.
export function insertApiQuery(
  api: Pick<
    InsertApi,
    "id" | "name" | "workspaceId" | "projectId" | "keyAuthId" | "authType" | "createdAtM"
  >,
) {
  return sql`INSERT INTO ${apis}
    (id, name, workspace_id, project_id, key_auth_id, auth_type, created_at_m, updated_at_m)
    VALUES (${api.id}, ${api.name}, ${api.workspaceId}, ${api.projectId},
      ${api.keyAuthId ?? null}, ${api.authType ?? null}, ${api.createdAtM ?? Date.now()}, ${Date.now()})`;
}
