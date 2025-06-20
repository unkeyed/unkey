import { Client } from "@planetscale/database";
import { type Database, drizzle, schema } from "@unkey/db";
import type { Logger } from "@unkey/worker-logging";
import { instrumentedFetch } from "./util/instrument-fetch";

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
      fetch: async (url: string, init?: RequestInit): Promise<Response> => {
        const { cache, ...initWithoutCache } = init || {};

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
          return instrumentedFetch()(u, initWithoutCache).catch((err) => {
            opts.logger.error("fetching from planetscale failed", {
              message: err instanceof Error ? err.message : String(err),
              retries: "disabled",
            });
            throw err;
          });
        }

        let lastError: Error | undefined = undefined;
        for (let i = 0; i <= opts.retry; i++) {
          try {
            return instrumentedFetch()(u, initWithoutCache);
          } catch (e) {
            lastError = e instanceof Error ? e : new Error(String(e));
            opts.logger.warn("fetching from planetscale failed", {
              url: u.toString(),
              attempt: i + 1,
              query: initWithoutCache.body,
              message: lastError.message,
            });
          }
        }

        opts.logger.error("fetching from planetscale failed", {
          message: lastError?.message,
          retries: "exhausted",
        });
        throw lastError;
      },
    }),
    {
      schema,
    },
  );
}

export * from "@unkey/db";
