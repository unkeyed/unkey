import { initTRPC } from "@trpc/server";
import { asc, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { expect, it, vi } from "vitest";
import {
  connectionTestDatabase,
  deploymentSeed,
} from "../../../../../../../internal/db/src/testing";
import { createAppConnection } from "./create";
import { deleteAppConnection } from "./delete";
import { listAppConnections } from "./list";
import { listRetainedAppConnections } from "./retained";
import { updateAppConnection } from "./update";

const queries = vi.hoisted(() => ({
  select: vi.fn(),
  transaction: vi.fn(),
  app: vi.fn(),
  connection: vi.fn(),
  connections: vi.fn(),
  environment: vi.fn(),
  project: vi.fn(),
  audit: vi.fn(),
}));
vi.mock("@/lib/db", async () => ({
  ...(await import("@unkey/db")),
  db: {
    select: queries.select,
    transaction: queries.transaction,
    query: {
      apps: { findFirst: queries.app },
      appConnections: { findFirst: queries.connection, findMany: queries.connections },
      environments: { findFirst: queries.environment },
      projects: { findFirst: queries.project },
    },
  },
}));
vi.mock("./write-helpers", async (importOriginal) => ({
  ...(await importOriginal<typeof import("./write-helpers")>()),
  audit: queries.audit,
}));
vi.mock("@/lib/trpc/trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  const t = initTRPC.context<{ workspace: { id: string } }>().create();
  return {
    workspaceProcedure: t.procedure,
    ratelimit: { read: "read", create: "create", update: "update", delete: "delete" },
    withRatelimit: () => t.middleware(({ next }) => next()),
  };
});

const databaseUrl = process.env.CONNECTION_TARGETS_TEST_DATABASE_URL;

it.skipIf(!databaseUrl)(
  "writes defaults and target settings atomically without changing saved deployment rules",
  async () => {
    if (!databaseUrl) {
      throw new Error("CONNECTION_TARGETS_TEST_DATABASE_URL is not set");
    }
    const database = await connectionTestDatabase(databaseUrl);
    const workspaceId = newId("workspace");
    const projectId = newId("project");
    const web = newId("app");
    const api = newId("app");
    const prod = newId("environment");
    const apiPreview = newId("environment");
    const binding = newId("connection");
    const one = newId("test");
    const two = newId("test");
    const apiPinned = newId("test");
    const tenant = { workspaceId, projectId };
    queries.select.mockImplementation(database.select.bind(database));
    queries.transaction.mockImplementation(database.transaction.bind(database));
    queries.app.mockImplementation(database.query.apps.findFirst.bind(database.query.apps));
    queries.connection.mockImplementation(
      database.query.appConnections.findFirst.bind(database.query.appConnections),
    );
    queries.connections.mockImplementation(
      database.query.appConnections.findMany.bind(database.query.appConnections),
    );
    queries.environment.mockImplementation(
      database.query.environments.findFirst.bind(database.query.environments),
    );
    queries.project.mockImplementation(
      database.query.projects.findFirst.bind(database.query.projects),
    );
    await database
      .insert(schema.projects)
      .values({ id: projectId, workspaceId, name: "Project", slug: "project" });
    await database.insert(schema.apps).values([
      { ...tenant, id: web, name: "Web", slug: "web" },
      { ...tenant, id: api, name: "API", slug: "api" },
    ]);
    await database.insert(schema.environments).values([
      { ...tenant, id: prod, appId: web, slug: "production", kind: "production" },
      { ...tenant, id: apiPreview, appId: api, slug: "preview" },
    ]);
    const rule = {
      ...tenant,
      appId: web,
      environmentId: prod,
      resourceType: "app",
      resourceId: api,
      name: "api",
    };
    await database.insert(schema.appConnections).values({ ...rule, id: binding });
    await database
      .insert(schema.connectionAppTargets)
      .values({ connectionId: binding, selectionMode: "automatic" });
    await database
      .insert(schema.deployments)
      .values(
        [one, two].map((id) => deploymentSeed({ ...tenant, id, appId: web, environmentId: prod })),
      );
    await database.insert(schema.deploymentConnections).values([
      { ...rule, deploymentId: one, connectionId: binding, createdAt: 1 },
      { ...rule, deploymentId: two, connectionId: binding, name: "renamed-api", createdAt: 2 },
    ]);
    await database.insert(schema.deploymentConnectionAppTargets).values([
      {
        deploymentId: one,
        connectionId: binding,
        selectionMode: "deployment",
        targetDeploymentId: apiPinned,
      },
      {
        deploymentId: two,
        connectionId: binding,
        selectionMode: "environment",
        targetEnvironmentId: apiPreview,
      },
    ]);

    const t = initTRPC.context<{ workspace: { id: string } }>().create();
    const router = t.router({
      create: createAppConnection,
      update: updateAppConnection,
      remove: deleteAppConnection,
      list: listAppConnections,
      retained: listRetainedAppConnections,
    });
    const caller = router.createCaller({ workspace: { id: workspaceId } });
    const input = { projectId, id: binding };
    const scope = { projectId, appId: web, environmentId: prod };
    const updated = {
      ...input,
      name: "custom-api",
      targetType: "environment" as const,
      targetEnvironmentId: apiPreview,
    };
    queries.audit.mockRejectedValueOnce(new Error("audit failed"));
    await expect(caller.update(updated)).rejects.toThrow("audit failed");
    expect(await caller.list(scope)).toEqual([
      expect.objectContaining({ name: "api", targetType: "automatic", targetEnvironmentId: null }),
    ]);
    await caller.update(updated);
    expect(await caller.list(scope)).toEqual([
      expect.objectContaining({
        name: "custom-api",
        targetType: "environment",
        targetEnvironmentId: apiPreview,
      }),
    ]);
    await caller.update({ ...input, targetType: "automatic" });
    expect(await caller.list(scope)).toEqual([
      expect.objectContaining({
        name: "custom-api",
        targetType: "automatic",
        targetEnvironmentId: null,
        targetDeploymentId: null,
      }),
    ]);
    expect(await caller.retained(scope)).toEqual([]);
    await expect(
      router.createCaller({ workspace: { id: newId("workspace") } }).remove(input),
    ).rejects.toMatchObject({ code: "NOT_FOUND" });
    await expect(caller.remove({ ...input, projectId: newId("project") })).rejects.toMatchObject({
      code: "NOT_FOUND",
    });
    queries.audit.mockRejectedValueOnce(new Error("audit failed"));
    await expect(caller.remove(input)).rejects.toThrow("audit failed");
    expect(await caller.retained(scope)).toEqual([]);
    expect(await caller.list(scope)).toHaveLength(1);
    await caller.remove(input);
    expect(await caller.list(scope)).toEqual([]);
    const targets = await database.select().from(schema.connectionAppTargets);
    expect(targets).toEqual([]);
    expect(await caller.retained(scope)).toEqual([
      { id: binding, targetAppName: "API", deployments: 2 },
    ]);
    expect(await caller.retained({ ...scope, environmentId: newId("environment") })).toEqual([]);
    await expect(
      router.createCaller({ workspace: { id: newId("workspace") } }).retained(scope),
    ).rejects.toMatchObject({ code: "NOT_FOUND" });
    await expect(caller.remove(input)).rejects.toMatchObject({ code: "NOT_FOUND" });
    const snapshots = await database
      .select({ name: schema.deploymentConnections.name })
      .from(schema.deploymentConnections)
      .orderBy(asc(schema.deploymentConnections.createdAt));
    expect(snapshots).toEqual([{ name: "api" }, { name: "renamed-api" }]);
    const snapshotTargets = await database
      .select({
        deployment_id: schema.deploymentConnectionAppTargets.deploymentId,
        selection_mode: schema.deploymentConnectionAppTargets.selectionMode,
        target_environment_id: schema.deploymentConnectionAppTargets.targetEnvironmentId,
        target_deployment_id: schema.deploymentConnectionAppTargets.targetDeploymentId,
      })
      .from(schema.deploymentConnectionAppTargets)
      .orderBy(asc(schema.deploymentConnectionAppTargets.pk));
    expect(snapshotTargets).toEqual([
      {
        deployment_id: one,
        selection_mode: "deployment",
        target_environment_id: null,
        target_deployment_id: apiPinned,
      },
      {
        deployment_id: two,
        selection_mode: "environment",
        target_environment_id: apiPreview,
        target_deployment_id: null,
      },
    ]);

    const create = {
      ...scope,
      targetAppId: api,
      targetType: "environment" as const,
      targetEnvironmentId: apiPreview,
    };
    queries.audit.mockRejectedValueOnce(new Error("audit failed"));
    await expect(caller.create(create)).rejects.toThrow("audit failed");
    expect(await caller.list(scope)).toEqual([]);
    const failedTargets = await database.select().from(schema.connectionAppTargets);
    expect(failedTargets).toEqual([]);
    const created = await caller.create(create);
    expect(await caller.list(scope)).toEqual([
      expect.objectContaining({
        id: created.id,
        name: "api",
        targetType: "environment",
        targetEnvironmentId: apiPreview,
      }),
    ]);
  },
);
