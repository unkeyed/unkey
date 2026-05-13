import { dbEnv } from "@/lib/env";
import { drizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";

const { DATABASE_HOST, DATABASE_USERNAME, DATABASE_PASSWORD } = dbEnv();
const isLocal = DATABASE_HOST.includes("localhost") || DATABASE_HOST.includes("127.0.0.1");

const pool = mysql.createPool({
  host: DATABASE_HOST.split(":")[0],
  port: DATABASE_HOST.includes(":") ? Number(DATABASE_HOST.split(":")[1]) : 3306,
  user: DATABASE_USERNAME,
  password: DATABASE_PASSWORD,
  database: "unkey",
  connectionLimit: 10,
  enableKeepAlive: true,
  ...(isLocal ? {} : { ssl: { rejectUnauthorized: true } }),
});

export const db = drizzle(pool, { schema, mode: "default" });

export * from "@unkey/db";
