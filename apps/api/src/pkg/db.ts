import { type MySql2Database, drizzle as drizzleMysql } from "drizzle-orm/mysql2";
import {
  type PlanetScaleDatabase,
  drizzle as drizzlePlanetscale,
} from "drizzle-orm/planetscale-serverless";

import { connect } from "@planetscale/database";
import mysql from "mysql2";

import { schema } from "@unkey/db";

export type Database = PlanetScaleDatabase<typeof schema> | MySql2Database<typeof schema>;

type ConnectionOptions = {
  host: string;
  username: string;
  password: string;
};

export function createConnection(
  opts: ConnectionOptions,
  mode: "planetscale" | "mysql" = "planetscale",
): Database {
  switch (mode) {
    case "planetscale": {
      drizzlePlanetscale(
        connect({
          host: opts.host,
          username: opts.username,
          password: opts.password,

          fetch: (url: string, init: any) => {
            (init as any).cache = undefined; // Remove cache header
            return fetch(url, init);
          },
        }),
        {
          schema,
        },
      );
    }
    case "mysql": {
      return drizzleMysql(
        mysql.createConnection({
          host: opts.host,
          user: opts.username,
          password: opts.password,
        }),
        {
          schema,
          mode: "default",
        },
      );
    }
  }
}
export * from "@unkey/db";
export * from "drizzle-orm";
