import { drizzle } from "drizzle-orm/planetscale-serverless";

import { connect } from "@planetscale/database";
import { schema } from "@unkey/db";

export const createConnection = (opts: { host: string; username: string; password: string, cached?: boolean }) =>
  drizzle(
    connect({
      ...opts,
      // biome-ignore lint/suspicious/noExplicitAny: TODO
      fetch: opts.cached ? undefined : (url: string, init: any) => {
        // biome-ignore lint/suspicious/noExplicitAny: TODO
        (init as any).cache = undefined; // Remove cache header
        return fetch(url, init);
      },
    }),
    {
      schema,
    },
  );
export * from "@unkey/db";
export * from "drizzle-orm";
