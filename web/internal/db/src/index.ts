export * from "./types";
import type { ExtractTablesWithRelations } from "drizzle-orm";
import type { MySqlTransaction } from "drizzle-orm/mysql-core";
import type {
  MySql2Database,
  MySql2PreparedQueryHKT,
  MySql2QueryResultHKT,
} from "drizzle-orm/mysql2";
import * as schema from "./schema";
export { schema };
export * from "drizzle-orm";
export { drizzle } from "drizzle-orm/mysql2";

export type Database = MySql2Database<typeof schema>;

export type Transaction = MySqlTransaction<
  MySql2QueryResultHKT,
  MySql2PreparedQueryHKT,
  typeof schema,
  ExtractTablesWithRelations<typeof schema>
>;

export { DrizzleQueryError } from "drizzle-orm/errors";
