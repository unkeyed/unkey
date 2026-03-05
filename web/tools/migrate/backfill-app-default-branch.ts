import { mysqlDrizzle, schema, sql } from "@unkey/db";
import mysql from "mysql2/promise";

async function main() {
  if (!process.env.DRIZZLE_DATABASE_URL) {
    throw new Error("DRIZZLE_DATABASE_URL is not set");
  }

  const conn = await mysql.createConnection(process.env.DRIZZLE_DATABASE_URL);
  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  console.log("Copying default_branch from projects to apps...");
  const [result] = await db.execute(
    sql`UPDATE apps a
        INNER JOIN projects p ON p.id = a.project_id
        SET a.default_branch = p.default_branch
        WHERE p.default_branch IS NOT NULL
          AND p.default_branch != ''`,
  );

  console.log("Result:", result);
  console.log("Migration complete!");
  await conn.end();
}

main().catch((err) => {
  console.error(err);
  process.exit(1);
});
