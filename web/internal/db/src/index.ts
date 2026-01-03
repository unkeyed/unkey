export * from "./types";
import type { ExtractTablesWithRelations } from "drizzle-orm";
import type {
  PlanetScaleDatabase,
  PlanetScaleTransaction,
} from "drizzle-orm/planetscale-serverless";
import * as schema from "./schema";
export { schema };
export * from "drizzle-orm";
export { drizzle } from "drizzle-orm/planetscale-serverless";
export { drizzle as mysqlDrizzle } from "drizzle-orm/mysql2";

export type Database = PlanetScaleDatabase<typeof schema>;

export type Transaction = PlanetScaleTransaction<
  typeof schema,
  ExtractTablesWithRelations<typeof schema>
>;

export { DrizzleQueryError } from "drizzle-orm/errors";
