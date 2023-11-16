import { drizzle as drizzleMysql } from "drizzle-orm/mysql2";
import { drizzle as drizzlePS } from "drizzle-orm/planetscale-serverless";

import { dbEnv } from "@/lib/env";
import { Config, connect } from "@planetscale/database";
import { schema } from "@unkey/db";

import mysql from "mysql2/promise";

const { DATABASE_ADAPTER, DATABASE_HOST, DATABASE_USERNAME, DATABASE_PASSWORD, DATABASE_NAME } =
  dbEnv();

const commonConfig = {
  host: DATABASE_HOST,
  password: DATABASE_PASSWORD,
};

const planetScaleConfig: Config = {
  ...commonConfig,
  username: DATABASE_USERNAME,
  // biome-ignore lint/suspicious/noExplicitAny: <explanation>
  fetch: (url: string, init: any) => {
    // biome-ignore lint/suspicious/noExplicitAny: TODO
    (init as any).cache = undefined; // Remove cache header
    return fetch(url, init);
  },
};

const mySqlConfig: mysql.PoolOptions = {
  ...commonConfig,
  user: DATABASE_USERNAME,
  database: DATABASE_NAME,
};

export const db =
  DATABASE_ADAPTER === "planet-scale"
    ? drizzlePS(connect(planetScaleConfig), { schema })
    : drizzleMysql(mysql.createPool(mySqlConfig), {
        schema,
        mode: "default",
      });

export * from "@unkey/db";
export * from "drizzle-orm";
