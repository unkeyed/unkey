import { Client } from "@planetscale/database";
import { type Database, drizzle, schema } from "@unkey/db";

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

      fetch: (url: string, init: any) => {
        (init as any).cache = undefined; // Remove cache header
        const u = new URL(url);
        /**
         * Running workerd in docker caused an issue where it was trying to use https but
         * encountered an ssl version error
         *
         * This enforces the use of http
         */
        if (u.hostname === "planetscale" || u.host.includes("localhost")) {
          u.protocol = "http";
        }
        return fetch(url, init);
      },
    }),
    {
      schema,
    },
  );
}
export * from "@unkey/db";
