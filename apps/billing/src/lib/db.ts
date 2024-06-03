import { drizzle } from "drizzle-orm/planetscale-serverless";

import { env } from "@/lib/env";
import { Client } from "@planetscale/database";
import { schema } from "@unkey/db";

export const connectDatabase = () =>
  drizzle(
    new Client({
      host: env().DATABASE_HOST,
      username: env().DATABASE_USERNAME,
      password: env().DATABASE_PASSWORD,

      fetch: (url: string, init: any) => {
        (init as any).cache = undefined; // Remove cache header
        return fetch(url, init);
      },
    }),
    {
      schema,
    },
  );

export * from "@unkey/db";
