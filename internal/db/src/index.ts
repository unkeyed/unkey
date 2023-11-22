export * from "./types";
import type { PlanetScaleDatabase } from "drizzle-orm/planetscale-serverless";
import * as schema from "./schema";
export type Database = PlanetScaleDatabase<typeof schema>;
export { schema };
