import { bigint, int, mysqlTable, varchar } from "drizzle-orm/mysql-core";

export const replicacheServers = mysqlTable("replicache_server", {
  id: varchar("id", { length: 256 }).primaryKey(),
  version: int("version").notNull(),
});

// export const replicacheClientGroups = mysqlTable("replicache_client_groups", {
//   id: char("id", { length: 36 }).primaryKey(),
//   actor: json("actor"),
//   cvrVersion: int("cvr_version").notNull(),
//   clientVersion: int("client_version").notNull(),
// })

export const replicacheClients = mysqlTable("replicache_clients", {
  id: varchar("id", { length: 36 }).primaryKey(),
  lastMutationId: bigint("last_mutation_id", { mode: "number" }).notNull().default(0),
  clientGroupId: varchar("client_group_id", { length: 36 }).notNull(),
  version: int("version").notNull(),
});
