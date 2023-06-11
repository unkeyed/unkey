// db.ts
import { drizzle, type PlanetScaleDatabase } from "drizzle-orm/planetscale-serverless";

import { connect } from "@planetscale/database";
import * as schema from "./schema";
import { InferModel } from "drizzle-orm";

export const createDatabase = (c: {
  host: string;
  username: string;
  password: string;
}) =>
  drizzle(
    connect({
      ...c,
      fetch: (url: string, init: any) => {
        (init as any)["cache"] = undefined; // Remove cache header
        return fetch(url, init);
      },
    }),
    {
      schema,
    },
  );

export const db = createDatabase({
  host: process.env.DATABASE_HOST!,
  username: process.env.DATABASE_USERNAME!,
  password: process.env.DATABASE_PASSWORD!,
});
export type Database = PlanetScaleDatabase<typeof schema>;

export type Key = InferModel<typeof schema.keys>;
export type Api = InferModel<typeof schema.apis>;
export type Tenant = InferModel<typeof schema.tenants>;
