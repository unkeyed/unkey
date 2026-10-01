import { initTRPC } from "@trpc/server";
import { drizzle, schema } from "@unkey/db";
import mysql from "mysql2/promise";
import { expect, it, onTestFinished, vi } from "vitest";
import { listAppConnectionTargets } from "./targets";

const queries = vi.hoisted(() => ({ select: vi.fn(), findApp: vi.fn() }));
vi.mock("@/lib/db", async () => ({
  ...(await import("@unkey/db")),
  db: { select: queries.select, query: { apps: { findFirst: queries.findApp } } },
}));
vi.mock("@/lib/trpc/trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  const t = initTRPC.context<{ workspace: { id: string } }>().create();
  return {
    workspaceProcedure: t.procedure,
    ratelimit: { read: "read" },
    withRatelimit: () => t.middleware(({ next }) => next()),
  };
});

const databaseUrl = process.env.CONNECTION_TARGETS_TEST_DATABASE_URL;

it.skipIf(!databaseUrl)(
  "resolves an older live pin without other environments or invalid targets consuming its limit",
  async () => {
    if (!databaseUrl) {
      throw new Error("CONNECTION_TARGETS_TEST_DATABASE_URL is not set");
    }
    const url = new URL(databaseUrl);
    expect(["localhost", "127.0.0.1", "::1"]).toContain(url.hostname);
    const name = `connection_targets_${process.pid}`;
    const admin = mysql.createPool(databaseUrl);
    await admin.query(`CREATE DATABASE \`${name}\``);
    url.pathname = `/${name}`;
    const pool = mysql.createPool(url.toString());
    onTestFinished(async () => {
      await pool.end();
      await admin.query(`DROP DATABASE IF EXISTS \`${name}\``);
      await admin.end();
    });
    const database = drizzle(pool, { schema, mode: "default" });
    queries.select.mockImplementation(database.select.bind(database));
    queries.findApp.mockImplementation(database.query.apps.findFirst.bind(database.query.apps));

    await pool.query(`CREATE TABLE apps (
      id VARCHAR(64) PRIMARY KEY, workspace_id VARCHAR(64), project_id VARCHAR(64),
      name VARCHAR(256), slug VARCHAR(256)
    )`);
    await pool.query(`CREATE TABLE environments (
      id VARCHAR(64) PRIMARY KEY, workspace_id VARCHAR(64), project_id VARCHAR(64),
      app_id VARCHAR(64), slug VARCHAR(256), kind VARCHAR(32)
    )`);
    await pool.query(`CREATE TABLE app_connections (
      id VARCHAR(64) PRIMARY KEY, workspace_id VARCHAR(64), project_id VARCHAR(64),
      app_id VARCHAR(64), environment_id VARCHAR(64), resource_type VARCHAR(32),
      resource_id VARCHAR(64), name VARCHAR(63), selection_mode VARCHAR(32),
      target_deployment_id VARCHAR(64),
      UNIQUE KEY app_name (app_id, environment_id, name),
      UNIQUE KEY app_resource (app_id, environment_id, resource_type, resource_id)
    )`);
    await pool.query(`CREATE TABLE deployments (
      id VARCHAR(64) PRIMARY KEY, workspace_id VARCHAR(64), project_id VARCHAR(64),
      app_id VARCHAR(64), environment_id VARCHAR(64), status VARCHAR(32),
      git_branch VARCHAR(256), image_resolved VARCHAR(512),
      created_at BIGINT, first_ready_at BIGINT
    )`);
    await pool.query(`INSERT INTO apps VALUES
      ('web', 'workspace', 'project', 'Web', 'web'),
      ('api', 'workspace', 'project', 'API', 'api'),
      ('foreign', 'other-workspace', 'project', 'Foreign', 'foreign')`);
    await pool.query(`INSERT INTO environments VALUES
      ('preview', 'workspace', 'project', 'web', 'preview', 'preview')`);
    const otherEnvironments = Array.from({ length: 501 }, (_, i) => [
      `other-${i}`,
      "workspace",
      "project",
      "web",
      `other-${i}`,
      "app",
      "api",
      `a-other-${i}`,
      "deployment",
      "other-pin",
    ]);
    const invalidTargets = [
      "web",
      "foreign",
      ...Array.from({ length: 501 }, (_, i) => `deleted-${i}`),
    ].map((id) => [
      `invalid-${id}`,
      "workspace",
      "project",
      "web",
      "preview",
      "app",
      id,
      `b-invalid-${id}`,
      "deployment",
      "other-pin",
    ]);
    await pool.query("INSERT INTO app_connections VALUES ?", [
      [
        ...otherEnvironments,
        ...invalidTargets,
        [
          "selected",
          "workspace",
          "project",
          "web",
          "preview",
          "app",
          "api",
          "z-selected",
          "deployment",
          "older-live-pin",
        ],
      ],
    ]);
    await pool.query("INSERT INTO deployments VALUES ?", [
      [
        ["older-live-pin", "workspace", "project", "api", "api-preview", "ready", null, null, 1, 1],
        ...Array.from({ length: 101 }, (_, i) => [
          `newer-${String(i).padStart(3, "0")}`,
          "workspace",
          "project",
          "api",
          "api-preview",
          "ready",
          null,
          null,
          100,
          100,
        ]),
      ],
    ]);

    const t = initTRPC.context<{ workspace: { id: string } }>().create();
    const caller = t
      .router({ targets: listAppConnectionTargets })
      .createCaller({ workspace: { id: "workspace" } });
    const input = {
      projectId: "project",
      appId: "web",
      environmentId: "preview",
      targetAppId: "api",
    };
    const first = await caller.targets(input);
    expect(first.deployments).toHaveLength(101);
    expect(first.deployments[0]).toMatchObject({ id: "older-live-pin", status: "ready" });
    expect(first.nextCursor).toEqual({ id: "newer-001", createdAt: 100 });
    const second = await caller.targets({ ...input, cursor: first.nextCursor });
    expect(second.deployments.map(({ id }) => id)).toEqual(["older-live-pin", "newer-000"]);
    expect(second.nextCursor).toBeNull();

    const other = await caller.targets({
      ...input,
      environmentId: "other-0",
      targetAppId: undefined,
    });
    expect(other.deployments).toEqual([]);
  },
);
