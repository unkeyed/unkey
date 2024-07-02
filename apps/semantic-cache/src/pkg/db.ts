import { Client } from "@planetscale/database";
import { type PlanetScaleDatabase, drizzle, schema } from "@unkey/db";
import type { Env } from "./env";
export type Database = PlanetScaleDatabase<typeof schema>;

export function createConnection(
  env: Pick<Env, "DATABASE_HOST" | "DATABASE_USERNAME" | "DATABASE_PASSWORD">,
): Database {
  return drizzle(
    new Client({
      host: env.DATABASE_HOST,
      username: env.DATABASE_USERNAME,
      password: env.DATABASE_PASSWORD,

      fetch: (url: string, init: any) => {
        (init as any).cache = undefined; // Remove cache header
        const u = new URL(url);
        // set protocol to http if localhost for CI testing
        if (u.host.includes("localhost")) {
          u.protocol = "http";
        }
        return fetch(u, init);
      },
    }),
    {
      schema,
    },
  );
}
export * from "@unkey/db";
