import { drizzle } from "drizzle-orm/mysql2";
import { createConnection } from "mysql2/promise";
import { expect, it } from "vitest";
import * as schema from "./index";

it.skipIf(!process.env.TEST_DATABASE_URL)(
  "resolves shared GitHub installations only within the connection workspace",
  async () => {
    const connection = await createConnection(process.env.TEST_DATABASE_URL ?? "");
    try {
      await connection.query(`CREATE TEMPORARY TABLE github_app_installations (
        pk BIGINT PRIMARY KEY,
        workspace_id VARCHAR(256) NOT NULL,
        installation_id BIGINT NOT NULL,
        UNIQUE (workspace_id, installation_id)
      )`);
      await connection.query(`CREATE TEMPORARY TABLE github_repo_connections (
        pk BIGINT PRIMARY KEY,
        workspace_id VARCHAR(256) NOT NULL,
        app_id VARCHAR(256) NOT NULL UNIQUE,
        installation_id BIGINT NOT NULL
      )`);
      await connection.query(`INSERT INTO github_app_installations VALUES
        (3, 'ws_other', 4701),
        (91, 'ws_target', 4701),
        (12, 'ws_target', 8802)`);
      await connection.query(`INSERT INTO github_repo_connections VALUES
        (8, 'ws_target', 'app_target', 4701),
        (22, 'ws_other', 'app_other', 4701),
        (45, 'ws_target', 'app_second', 8802)`);

      const db = drizzle(connection, { schema, mode: "planetscale" });
      const connections = await db.query.githubRepoConnections.findMany({
        columns: { appId: true },
        with: {
          installation: {
            columns: { pk: true, workspaceId: true, installationId: true },
          },
        },
        orderBy: (table, { asc }) => asc(table.pk),
      });

      expect(connections).toEqual([
        {
          appId: "app_target",
          installation: { pk: 91, workspaceId: "ws_target", installationId: 4701 },
        },
        {
          appId: "app_other",
          installation: { pk: 3, workspaceId: "ws_other", installationId: 4701 },
        },
        {
          appId: "app_second",
          installation: { pk: 12, workspaceId: "ws_target", installationId: 8802 },
        },
      ]);
    } finally {
      await connection.end();
    }
  },
);
