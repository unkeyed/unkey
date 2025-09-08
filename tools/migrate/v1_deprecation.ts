import { ClickHouse } from "@unkey/clickhouse";
import { mysqlDrizzle, schema } from "@unkey/db";
import { Resend } from "@unkey/resend";
import { WorkOS } from "@workos-inc/node";
import mysql from "mysql2/promise";
import { z } from "zod";
async function main() {
  const ch = new ClickHouse({
    url: process.env.CLICKHOUSE_URL,
  });

  const conn = await mysql.createConnection(
    `mysql://${process.env.DATABASE_USERNAME}:${process.env.DATABASE_PASSWORD}@${process.env.DATABASE_HOST}:3306/unkey?ssl={}`,
  );

  await conn.ping();
  const db = mysqlDrizzle(conn, { schema, mode: "default" });

  const workos = new WorkOS(process.env.WORKOS_API_KEY);

  const resend = new Resend({
    apiKey: process.env.RESEND_API_KEY,
  });
  console.log("starting");

  const rows = await ch.querier.query({
    query: `
    SELECT
      workspace_id,
      splitByChar('?', path, 1)[1] as path
    FROM metrics.api_requests_per_day_v1
    WHERE startsWith(path, '/v1/')
    AND workspace_id != ''
    AND time >= (now() - INTERVAL 30 DAY)
    GROUP BY workspace_id, path`,
    schema: z.object({
      workspace_id: z.string(),
      path: z.string(),
    }),
  })({});
  if (rows.err) {
    console.error(rows.err);
    process.exit(1);
  }
  console.log(`Found ${new Set(rows.val.map(r => r.workspace_id)).size} workspaces across ${rows.val.length} paths`);
  const workspaceToPaths = new Map<string, string[]>();
  for (const row of rows.val) {
    const paths = workspaceToPaths.get(row.workspace_id) || [];
    paths.push(row.path);
    workspaceToPaths.set(row.workspace_id, paths);
  }

  for (const [workspaceId, paths] of workspaceToPaths.entries()) {
    console.log(workspaceId, paths);

    const workspace = await db.query.workspaces.findFirst({
      where: (table, { eq }) => eq(table.id, workspaceId),
    });
    if (!workspace) {
      console.error(`Workspace ${workspaceId} not found`);
      continue;
    }

    console.log(workspace.name);

    const members = await workos.userManagement.listOrganizationMemberships({
      organizationId: workspace.orgId,
    });

    const users = await Promise.all(
      members.data.map(async (member) => workos.userManagement.getUser(member.userId)),
    );

    for (const user of users) {
      console.log(user.email);

      if (user.email === "andreas@unkey.com") {
        await resend.sendApiV1MigrationEmail({
          email: user.email,
          name: user.firstName,
          workspace: workspace.name,
          deprecatedEndpoints: paths,
        });
        throw new Error(`User ${user.email} is from unkey.com`);
      }
    }
  }
}

main();
