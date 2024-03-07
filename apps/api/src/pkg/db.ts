import { type PlanetScaleDatabase, drizzle } from "drizzle-orm/planetscale-serverless";

import { AsyncLocalStorage } from "node:async_hooks";
import { Client } from "@planetscale/database";
import { schema } from "@unkey/db";
export type Database = PlanetScaleDatabase<typeof schema>;

type ConnectionOptions = {
  host: string;
  username: string;
  password: string;
};

export function createConnection(opts: ConnectionOptions): Database {
  return drizzle(
    new Client({
      host: opts.host,
      username: opts.username,
      password: opts.password,

      /**
       * AsyncLocalStorage is required to preserve otel trace context
       *
       * If we don't bind it, multiple traces are generated instead of a single one
       *
       * Cloudflare is addressing this in the runtime already
       */
      fetch: AsyncLocalStorage.bind((url: string, init: any) => {
        (init as any).cache = undefined; // Remove cache header
        const u = new URL(url);
        // set protocol to http if localhost for CI testing
        if (u.host.includes("localhost")) {
          u.protocol = "http";
        }
        return fetch(u, init);
      }),
    }),
    {
      schema,
    },
  );
}
export * from "@unkey/db";
