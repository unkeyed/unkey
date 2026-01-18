import { bigint, index, mysqlEnum, mysqlTable, varchar } from "drizzle-orm/mysql-core";
import { longblob } from "./util/longblob";

export const stateChanges = mysqlTable(
  "state_changes",
  {
    sequence: bigint("sequence", { mode: "number", unsigned: true }).autoincrement().primaryKey(),

    // The apply or delete protobuf blob

    resourceType: mysqlEnum("resource_type", ["sentinel", "deployment"]).notNull(),
    state: longblob("state").notNull(),

    clusterId: varchar("cluster_id", { length: 256 }).notNull(),

    createdAt: bigint("created_at", {
      mode: "number",
      unsigned: true,
    }).notNull(),
  },
  (table) => [index("cluster_id_sequence").on(table.clusterId, table.sequence)],
);
