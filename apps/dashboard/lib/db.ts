import { dbEnv } from "@/lib/env";
import { Client } from "@planetscale/database";
import { drizzle, schema } from "@unkey/db";

export const db = drizzle(
  new Client({
    host: dbEnv().DATABASE_HOST,
    username: dbEnv().DATABASE_USERNAME,
    password: dbEnv().DATABASE_PASSWORD,

    // biome-ignore lint/suspicious/noExplicitAny: safe to leave
    fetch: (url: string, init: any) => {
      // biome-ignore lint/suspicious/noExplicitAny: safe to leave
      (init as any).cache = undefined; // Remove cache header
      const u = new URL(url);
      // set protocol to http if localhost or docker planetscale service for CI testing
      if (u.host.includes("localhost") || u.host === "planetscale:3900") {
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
