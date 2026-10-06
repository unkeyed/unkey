import { bigint, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

// One row per WorkOS agent registration. The unique registration id is the
// one-workspace-per-claim lock: a second insert fails even when two requests
// race. root_key_id stays null until the single first-root-key issuance
// commits, and that issuance holds this row for update.
export const agentSignups = mysqlTable("agent_signups", {
  pk: primaryKey(),
  id: id("id").notNull().unique(),
  agentRegistrationId: caseSensitiveVarchar("agent_registration_id", { length: 256 })
    .notNull()
    .unique(),
  workosUserId: caseSensitiveVarchar("workos_user_id", { length: 256 }).notNull(),
  workspaceId: id("workspace_id"),
  rootKeyId: id("root_key_id"),
  status: varchar("status", { length: 32 }).notNull(),
  requestedPermissions: json("requested_permissions").$type<string[]>(),
  createdAtM: bigint("created_at_m", { mode: "number" }).notNull(),
  updatedAtM: bigint("updated_at_m", { mode: "number" }),
});
