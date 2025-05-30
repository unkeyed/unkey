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
      fetch: (url: string, init?: RequestInit) => {
        if (init) {
          // Remove cache header
          const { cache, ...restInit } = init;
          const modifiedInit = restInit;

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
          return fetch(u.toString(), modifiedInit);
        }

        return fetch(url);
      },
    }),
    {
      schema,
    },
  );
}

export * from "@unkey/db";
