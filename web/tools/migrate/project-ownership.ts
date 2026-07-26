import { newId } from "@unkey/id";
import mysql, { type ResultSetHeader, type RowDataPacket } from "mysql2/promise";

interface WorkspaceRow extends RowDataPacket {
  id: string;
}

interface ProjectRow extends RowDataPacket {
  id: string;
}

async function main() {
  const databaseUrl = process.env.DRIZZLE_DATABASE_URL;
  if (!databaseUrl) {
    throw new Error("DRIZZLE_DATABASE_URL is not set");
  }

  const connection = await mysql.createConnection(databaseUrl);

  try {
    await connection.ping();

    const [workspaces] = await connection.query<WorkspaceRow[]>(
      "SELECT id FROM workspaces ORDER BY pk",
    );
    console.info(`Found ${workspaces.length} workspaces`);

    let failures = 0;

    for (const workspace of workspaces) {
      try {
        const [projects] = await connection.execute<ProjectRow[]>(
          "SELECT id FROM projects WHERE workspace_id = ? AND BINARY slug = 'default' LIMIT 1",
          [workspace.id],
        );

        let projectId = projects[0]?.id;
        if (!projectId) {
          projectId = newId("project");
          await connection.execute<ResultSetHeader>(
            "INSERT INTO projects (id, workspace_id, name, slug, delete_protection, created_at) VALUES (?, ?, 'Default', 'default', true, ?)",
            [projectId, workspace.id, Date.now()],
          );
          console.info(`Created default project ${projectId} for workspace ${workspace.id}`);
        }

        const [apis] = await connection.execute<ResultSetHeader>(
          "UPDATE apis SET project_id = ? WHERE workspace_id = ? AND project_id = ''",
          [projectId, workspace.id],
        );
        const [keyAuth] = await connection.execute<ResultSetHeader>(
          "UPDATE key_auth SET project_id = ? WHERE workspace_id = ? AND project_id = ''",
          [projectId, workspace.id],
        );
        const [identities] = await connection.execute<ResultSetHeader>(
          "UPDATE identities SET project_id = ? WHERE workspace_id = ? AND project_id = ''",
          [projectId, workspace.id],
        );
        const [roles] = await connection.execute<ResultSetHeader>(
          "UPDATE roles SET project_id = ? WHERE workspace_id = ? AND project_id = ''",
          [projectId, workspace.id],
        );
        const [permissions] = await connection.execute<ResultSetHeader>(
          "UPDATE permissions SET project_id = ? WHERE workspace_id = ? AND project_id = ''",
          [projectId, workspace.id],
        );
        const [ratelimitNamespaces] = await connection.execute<ResultSetHeader>(
          "UPDATE ratelimit_namespaces SET project_id = ? WHERE workspace_id = ? AND project_id = ''",
          [projectId, workspace.id],
        );

        console.info(`Backfilled workspace ${workspace.id}`, {
          apis: apis.affectedRows,
          keyAuth: keyAuth.affectedRows,
          identities: identities.affectedRows,
          roles: roles.affectedRows,
          permissions: permissions.affectedRows,
          ratelimitNamespaces: ratelimitNamespaces.affectedRows,
        });
      } catch (error) {
        failures++;
        console.error(`Failed to migrate workspace ${workspace.id}`, error);
      }
    }

    if (failures > 0) {
      throw new Error(`Failed to migrate ${failures} workspaces`);
    }

    console.info("Project ownership migration complete");
  } finally {
    await connection.end();
  }
}

main().catch((error: unknown) => {
  console.error("Project ownership migration failed", error);
  process.exitCode = 1;
});
