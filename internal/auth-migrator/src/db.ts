import { Client } from "@planetscale/database";
import { drizzle, schema, type Database } from "@unkey/db";

export const db: Database = drizzle(
  new Client({
    host: process.env.DATABASE_HOST,
    username: process.env.DATABASE_USERNAME,
    password: process.env.DATABASE_PASSWORD,

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
