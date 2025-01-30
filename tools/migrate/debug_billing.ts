import { mysqlDrizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";

async function main() {
  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  let workspaces = await db.query.workspaces.findMany({
    where: (table, { isNotNull, isNull, not, eq, and }) =>
      and(
        isNotNull(table.stripeCustomerId),
        isNotNull(table.subscriptions),
        not(eq(table.plan, "free")),
        isNull(table.deletedAt),
      ),
  });
  // hack to filter out workspaces with `{}` as subscriptions
  workspaces = workspaces.filter(
    (ws) => ws.subscriptions && Object.keys(ws.subscriptions).length > 0,
  );

  console.info(`found ${workspaces.length} workspaces`);

  console.info(workspaces.map((ws) => ({ name: ws.name, id: ws.id })));
}

main().then(() => process.exit(0));
