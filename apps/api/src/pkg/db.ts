import { type PlanetScaleDatabase, drizzle } from "drizzle-orm/planetscale-serverless";

import { Connection, connect } from "@planetscale/database";

import { schema } from "@unkey/db";

export type Database = PlanetScaleDatabase<typeof schema>;

type ConnectionOptions = {
  host: string;
  username: string;
  password: string;
};

export function createConnection(opts: ConnectionOptions): { db: Database; rawDB: Connection } {
  return {
    db: drizzle(
      connect({
        host: opts.host,
        username: opts.username,
        password: opts.password,

        fetch: (url: string, init: any) => {
          delete init.cache;
          return fetch(url, init);
        },
      }),
      {
        schema,
      },
    ),
    rawDB: connect({
      host: opts.host,
      username: opts.username,
      password: opts.password,

      fetch: (url: string, init: any) => {
        delete init.cache;
        return fetch(url, init);
      },
    }),
  };
}
export * from "@unkey/db";
