import { Client } from "@planetscale/database";
import { drizzle, schema } from "@unkey/db";
import { dbEnv } from "./env";

export const db = drizzle(
  new Client({
    host: dbEnv().DATABASE_HOST,
    username: dbEnv().DATABASE_USERNAME,
    password: dbEnv().DATABASE_PASSWORD,

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

export * from "@unkey/db";
