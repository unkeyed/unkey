import { int, mysqlTable, varchar } from "drizzle-orm/mysql-core";

export const replicacheServers = mysqlTable("replicache_server", {
  id: varchar("id", { length: 256 }).primaryKey(),
  version: int("version").notNull(),
});

export const replicacheClients = mysqlTable("replicache_clients", {
  id: varchar("id", { length: 256 }).primaryKey(),
  clientGroupId: varchar("client_group_id", { length: 256 }).notNull(),
  lastMutationId: int("last_mutation_id").notNull(),
  version: int("version").notNull(),
});
