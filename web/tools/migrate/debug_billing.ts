import { drizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";

async function main() {
  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );

  await conn.ping();
  const db = drizzle(conn, { schema, mode: "default" });

  // The stripe customer id and tier moved to workspace_billing, so the only
  // discriminator left on the workspaces row is the legacy subscriptions blob.
  let workspaces = await db.query.workspaces.findMany({
    where: (table, { isNotNull, isNull, and }) =>
      and(isNotNull(table.subscriptions), isNull(table.deletedAtM)),
  });
  // hack to filter out workspaces with `{}` as subscriptions
  workspaces = workspaces.filter(
    (ws) => ws.subscriptions && Object.keys(ws.subscriptions).length > 0,
  );

  console.info(`found ${workspaces.length} workspaces`);

  console.info(workspaces.map((ws) => ({ name: ws.name, id: ws.id })));
}

main().then(() => process.exit(0));
