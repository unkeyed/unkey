import { dbEnv } from "@/lib/env";
import { createCommentedPool, drizzle, schema, staticTagsFromEnv } from "@unkey/db";

const { DATABASE_HOST, DATABASE_USERNAME, DATABASE_PASSWORD } = dbEnv();
const isLocal = DATABASE_HOST.includes("localhost") || DATABASE_HOST.includes("127.0.0.1");

const pool = createCommentedPool(
  {
    host: DATABASE_HOST.split(":")[0],
    port: DATABASE_HOST.includes(":") ? Number(DATABASE_HOST.split(":")[1]) : 3306,
    user: DATABASE_USERNAME,
    password: DATABASE_PASSWORD,
    database: "unkey",
    connectionLimit: 10,
    enableKeepAlive: true,
    ...(isLocal ? {} : { ssl: { rejectUnauthorized: true } }),
  },
  staticTagsFromEnv("dashboard"),
);

export const db = drizzle(pool, { schema, mode: "default" });

export * from "@unkey/db";
