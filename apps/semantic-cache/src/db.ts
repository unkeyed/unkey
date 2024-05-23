import { type PlanetScaleDatabase, drizzle } from "drizzle-orm/planetscale-serverless";

import { Client } from "@planetscale/database";
import { schema } from "@unkey/db";
export type Database = PlanetScaleDatabase<typeof schema>;

type ConnectionOptions = {
  host: string;
  username: string;
  password: string;
};

export function createConnection(opts: ConnectionOptions): Database {
  return drizzle(
    new Client({
      host: opts.host,
      username: opts.username,
      password: opts.password,
    }),
    {
      schema,
    },
  );
}
export * from "@unkey/db";
