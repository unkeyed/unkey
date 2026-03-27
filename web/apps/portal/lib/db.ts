import { env } from "@/lib/env";
import { Client } from "@planetscale/database";
import { type Database, drizzle, schema } from "@unkey/db";

/**
 * Lazily initialized database connection.
 * Deferred so that `next build` can compile pages without requiring
 * DATABASE_* env vars at build time (they're only needed at runtime).
 */
let _db: Database | null = null;

export function getDb(): Database {
  if (!_db) {
    _db = drizzle(
      new Client({
        host: env().DATABASE_HOST,
        username: env().DATABASE_USERNAME,
        password: env().DATABASE_PASSWORD,
        // biome-ignore lint/suspicious/noExplicitAny: PlanetScale Fetch type is incompatible with standard RequestInit
        fetch: (url: string, init: any) => {
          const requestInit = init as RequestInit & { cache?: string };
          requestInit.cache = undefined;
          const u = new URL(url);
          if (u.host.includes("localhost") || u.host === "planetscale:3900") {
            u.protocol = "http";
          }
          return fetch(u, requestInit);
        },
      }),
      { schema },
    );
  }
  return _db;
}

/**
 * @deprecated Use `getDb()` instead. Kept as a convenience alias during migration.
 */
export const db = new Proxy({} as Database, {
  get(_target, prop, receiver) {
    return Reflect.get(getDb(), prop, receiver);
  },
});
