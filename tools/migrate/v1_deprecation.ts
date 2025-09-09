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
    AND workspace_id != 'ws_2vUFz88G6TuzMQHZaUhXADNyZWMy' // filter out special workspaces
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

  let emailsSent = 0;

  console.log(
    `Found ${new Set(rows.val.map((r) => r.workspace_id)).size} workspaces across ${rows.val.length} paths`,
  );
  const workspaceToPaths = new Map<string, string[]>();
  for (const row of rows.val) {
    if (row.workspace_id.startsWith("test_")) {
      continue;
    }
    const paths = workspaceToPaths.get(row.workspace_id) || [];
    paths.push(row.path);
    workspaceToPaths.set(row.workspace_id, paths);
  }

  for (const [workspaceId, paths] of workspaceToPaths.entries()) {
    if (paths.includes("/v1/analytics.getVerifications")) {
      console.warn(
        `Skipping workspace ${workspaceId} due to analytics endpoint: ${paths.join(", ")}`,
      );
      continue;
    }
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
      limit: 100,
    });

    for (const member of members.data) {
      await new Promise((resolve) => setTimeout(resolve, 500));
      const user = await workos.userManagement.getUser(member.userId);
      console.log(`User: ${user.email}`);
      await resend.sendApiV1MigrationEmail({
        email: user.email,
        name: user.firstName,
        workspace: workspace.name,
        deprecatedEndpoints: paths,
      });
      emailsSent++;
    }
  }

  console.info(`Emails sent: ${emailsSent}`);
}

main();
