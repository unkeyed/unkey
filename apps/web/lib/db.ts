// db.ts

import { connect } from "@planetscale/database";
import { schema } from "@unkey/db";
import { drizzle } from "drizzle-orm/planetscale-serverless";
import { env } from "./env";

export * from "@unkey/db";
export const db = drizzle(
  connect({
    host: env.DATABASE_HOST,
    username: env.DATABASE_USERNAME,
    password: env.DATABASE_PASSWORD,

    fetch: (url: string, init: any) => {
      (init as any)["cache"] = undefined; // Remove cache header
      return fetch(url, init);
    },
  }),
  {
    schema,
  },
);
