import { env } from "@/lib/env";
import { Client } from "@planetscale/database";
import { drizzle, schema } from "@unkey/db";

export const db = drizzle(
  new Client({
    host: env().DATABASE_HOST,
    username: env().DATABASE_USERNAME,
    password: env().DATABASE_PASSWORD,
    fetch: (url: string, init: RequestInit) => {
      const requestInit = { ...init, cache: undefined };
      const u = new URL(url);
      if (u.host.includes("localhost") || u.host === "planetscale:3900") {
        u.protocol = "http";
      }
      return fetch(u, requestInit);
    },
  }),
  { schema },
);
