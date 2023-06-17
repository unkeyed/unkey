import { schema } from "@unkey/db";
import { env } from "./env";
import { drizzle } from "drizzle-orm/planetscale-serverless";
import { connect } from "@planetscale/database";

// create the connection
const connection = connect({
  host: env.DATABASE_HOST,
  username: env.DATABASE_USERNAME,
  password: env.DATABASE_PASSWORD,
});

export const db = drizzle(connection, { schema });
export type Database = typeof db;
export type { Key, Api, Workspace } from "@unkey/db";
