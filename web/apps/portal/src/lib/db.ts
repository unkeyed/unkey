import "@tanstack/react-start/server-only";
import { drizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";
import { env } from "./env";

let _pool: mysql.Pool | null = null;

function getPool() {
  if (!_pool) {
    const { DATABASE_HOST, DATABASE_USERNAME, DATABASE_PASSWORD } = env();
    const isLocal = DATABASE_HOST.includes("localhost") || DATABASE_HOST.includes("127.0.0.1");

    _pool = mysql.createPool({
      host: DATABASE_HOST.split(":")[0],
      port: DATABASE_HOST.includes(":") ? Number(DATABASE_HOST.split(":")[1]) : 3306,
      user: DATABASE_USERNAME,
      password: DATABASE_PASSWORD,
      database: "unkey",
      connectionLimit: 5,
      enableKeepAlive: true,
      ...(isLocal ? {} : { ssl: { rejectUnauthorized: true } }),
    });
  }
  return _pool;
}

export const db = drizzle(getPool(), { schema, mode: "default" });

export { schema };
