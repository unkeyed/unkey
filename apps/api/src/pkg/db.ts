import { Client } from "@planetscale/database";
import { type PlanetScaleDatabase, drizzle, schema } from "@unkey/db";
import type { Logger } from "@unkey/worker-logging";
export type Database = PlanetScaleDatabase<typeof schema>;

type ConnectionOptions = {
  host: string;
  username: string;
  password: string;
  retry: number | false;
  logger: Logger;
};

export function createConnection(opts: ConnectionOptions): Database {
  return drizzle(
    new Client({
      host: opts.host,
      username: opts.username,
      password: opts.password,

      fetch: (url: string, init: any) => {
        (init as any).cache = undefined; // Remove cache header
        const u = new URL(url);
        /**
         * Running workerd in docker caused an issue where it was trying to use https but
         * encountered an ssl version error
         *
         * This enforces the use of http
         */
        if (u.hostname === "planetscale" || u.host.includes("localhost")) {
          u.protocol = "http";
        }

        if (!opts.retry) {
          return fetch(u, init).catch((err) => {
            opts.logger.error("fetching from planetscale failed", {
              message: (err as Error).message,
              retries: "disabled",
            });
            throw err;
          });
        }

        let err: Error | undefined = undefined;
        for (let i = 0; i <= opts.retry; i++) {
          try {
            return fetch(u, init);
          } catch (e) {
            err = e as Error;
            opts.logger?.warn("fetching from planetscale failed", {
              url: u.toString(),
              attempt: i + 1,
              query: init.body,
              message: err.message,
            });
          }
        }
        opts.logger.error("fetching from planetscale failed", {
          message: err!.message,
          retries: "exhausted",
        });
        throw err;
      },
    }),
    {
      schema,
    },
  );
}
export * from "@unkey/db";
