import { schema } from "@unkey/db";
import { env } from "./env";
import { drizzle } from "drizzle-orm/mysql2";
import { createConnection } from "mysql2/promise";

export async function initDB() {
  const connection = await createConnection(env.DATABASE_URL);

  return drizzle(connection, { schema });
}
export type Database = Awaited<ReturnType<typeof initDB>>;
export type { Key, Api, Workspace } from "@unkey/db";
