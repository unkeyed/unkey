import { bigint, json, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { caseSensitiveVarchar } from "./util/case_sensitive_varchar";
import { id } from "./util/id";
import { primaryKey } from "./util/primary_key";

// RFC 7591 public clients. Rows are never updated or deleted because a client
// never learns that its client_id expired. metadata_hash dedupes repeated
// registrations of the same name and redirect URIs.
export const oauthClients = mysqlTable("oauth_clients", {
  pk: primaryKey(),
  id: id("id").notNull().unique(),
  name: varchar("name", { length: 256 }).notNull(),
  redirectUris: json("redirect_uris").$type<string[]>().notNull(),
  metadataHash: caseSensitiveVarchar("metadata_hash", { length: 64 }).notNull().unique(),
  registrarIp: varchar("registrar_ip", { length: 64 }),
  registrarUserAgent: varchar("registrar_user_agent", { length: 512 }),
  createdAt: bigint("created_at", { mode: "number" }).notNull(),
});
