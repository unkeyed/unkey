import { drizzle } from "drizzle-orm/planetscale-serverless";

import { dbEnv } from "@/lib/env";
import { connect } from "@planetscale/database";
import { schema } from "@unkey/db";

export const db = drizzle(
  connect({
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
