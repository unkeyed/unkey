import { newId } from "@unkey/id";
import mysql, { type ResultSetHeader, type RowDataPacket } from "mysql2/promise";

const databaseURL = process.env.DRIZZLE_DATABASE_URL;
if (!databaseURL) {
  throw new Error("DRIZZLE_DATABASE_URL is not set");
}

async function main(databaseURL: string): Promise<void> {
  const connection = await mysql.createConnection(databaseURL);

  try {
    const [workspaces] = await connection.query<(RowDataPacket & { id: string })[]>(
      "SELECT id FROM workspaces",
    );
    console.log(`Starting project ownership migration for ${workspaces.length} workspaces`);

    for (const [index, workspace] of workspaces.entries()) {
      console.log(`[${index + 1}/${workspaces.length}] Migrating workspace ${workspace.id}`);

      const [projects] = await connection.execute<(RowDataPacket & { id: string })[]>(
        "SELECT id FROM projects WHERE workspace_id = ? AND slug = 'default' LIMIT 1",
        [workspace.id],
      );
      let projectId = projects[0]?.id;

      if (projectId) {
        console.log(`Using existing default project ${projectId}`);
      } else {
        projectId = newId("project");
        await connection.execute(
          "INSERT INTO projects (id, workspace_id, name, slug, delete_protection, created_at) VALUES (?, ?, 'Default', 'default', true, ?)",
          [projectId, workspace.id, Date.now()],
        );
        console.log(`Created default project ${projectId}`);
      }

      const [apis] = await connection.execute<ResultSetHeader>(
        "UPDATE apis SET project_id = ? WHERE workspace_id = ? AND project_id = ''",
        [projectId, workspace.id],
      );
      console.log(`Updated ${apis.affectedRows} APIs`);

      const [keyAuth] = await connection.execute<ResultSetHeader>(
        "UPDATE key_auth SET project_id = ? WHERE workspace_id = ? AND project_id = ''",
        [projectId, workspace.id],
      );
      console.log(`Updated ${keyAuth.affectedRows} keyspaces`);

      const [identities] = await connection.execute<ResultSetHeader>(
        "UPDATE identities SET project_id = ? WHERE workspace_id = ? AND project_id = ''",
        [projectId, workspace.id],
      );
      console.log(`Updated ${identities.affectedRows} identities`);

      const [permissions] = await connection.execute<ResultSetHeader>(
        "UPDATE permissions SET project_id = ? WHERE workspace_id = ? AND project_id = ''",
        [projectId, workspace.id],
      );
      console.log(`Updated ${permissions.affectedRows} permissions`);

      const [roles] = await connection.execute<ResultSetHeader>(
        "UPDATE roles SET project_id = ? WHERE workspace_id = ? AND project_id = ''",
        [projectId, workspace.id],
      );
      console.log(`Updated ${roles.affectedRows} roles`);

      const [ratelimitNamespaces] = await connection.execute<ResultSetHeader>(
        "UPDATE ratelimit_namespaces SET project_id = ? WHERE workspace_id = ? AND project_id = ''",
        [projectId, workspace.id],
      );
      console.log(`Updated ${ratelimitNamespaces.affectedRows} rate limit namespaces`);
    }

    console.log("Project ownership migration complete");
  } finally {
    await connection.end();
  }
}

main(databaseURL).catch((error: unknown) => {
  console.error("Project ownership migration failed", error);
  process.exitCode = 1;
});
