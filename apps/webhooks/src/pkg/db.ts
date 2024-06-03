import { type PlanetScaleDatabase, drizzle } from "drizzle-orm/planetscale-serverless";

import { Client } from "@planetscale/database";
import { schema } from "@unkey/db";
export type Database = PlanetScaleDatabase<typeof schema>;

type ConnectionOptions = {
  DATABASE_HOST: string;
  DATABASE_USERNAME: string;
  DATABASE_PASSWORD: string;
};

export function createConnection(opts: ConnectionOptions): Database {
  return drizzle(
    new Client({
      host: opts.DATABASE_HOST,
      username: opts.DATABASE_USERNAME,
      password: opts.DATABASE_PASSWORD,

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
