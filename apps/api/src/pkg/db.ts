import { drizzle } from "drizzle-orm/planetscale-serverless";

import { connect } from "@planetscale/database";
import { schema } from "@unkey/db";

export const createConnection = (opts: Omit<Parameters<typeof connect>[0], "fetch">) =>
  drizzle(
    connect({
      ...opts,
      fetch: (url: string, init: any) => {
        (init as any).cache = undefined; // Remove cache header cause cf can't handle it
        return fetch(url, init);
      },
    }),
    {
      schema,
    },
  );
export * from "@unkey/db";
export * from "drizzle-orm";
