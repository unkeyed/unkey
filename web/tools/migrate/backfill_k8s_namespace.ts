import { mysqlDrizzle, schema } from "@unkey/db";
import { dns1035 } from "@unkey/id";
import { eq, isNull } from "drizzle-orm";
import mysql from "mysql2/promise";

async function main() {
  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  let cursor = "";
  let total = 0;

  do {
    const workspaces = await db.query.workspaces.findMany({
      where: (table, { and, gt }) => and(isNull(table.k8sNamespace), gt(table.id, cursor)),
      columns: { id: true },
      limit: 1000,
      orderBy: (table, { asc }) => asc(table.id),
    });

    cursor = workspaces.at(-1)?.id ?? "";

    for (const workspace of workspaces) {
      const namespace = dns1035();

      await db
        .update(schema.workspaces)
        .set({ k8sNamespace: namespace })
        .where(eq(schema.workspaces.id, workspace.id));

      total++;
      console.log(`updated ${workspace.id} -> ${namespace}`);
    }
  } while (cursor !== "");

  console.log(`done. backfilled ${total} workspaces`);
  await conn.end();
}

main();
