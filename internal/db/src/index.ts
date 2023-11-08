export * from "./types";
import * as schema from "./schema";
import type { PlanetScaleDatabase } from "drizzle-orm/planetscale-serverless"
export type Database = PlanetScaleDatabase<typeof schema>;
export {
  schema
}
