import { initTRPC } from "@trpc/server";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { expect, it, vi } from "vitest";
import {
  connectionTestDatabase,
  deploymentSeed,
} from "../../../../../../../internal/db/src/testing";
import { listDeploymentConnectionPins } from "./list-connection-pins";

const queries = vi.hoisted(() => ({ select: vi.fn() }));
vi.mock("@/lib/db", async () => ({
  ...(await import("@unkey/db")),
  db: { select: queries.select },
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

it.skipIf(!databaseUrl)("lists only live deployment pins in the requested tenant", async () => {
  if (!databaseUrl) {
    throw new Error("CONNECTION_TARGETS_TEST_DATABASE_URL is not set");
  }
  const database = await connectionTestDatabase(databaseUrl);
  queries.select.mockImplementation(database.select.bind(database));
  const workspaceId = newId("workspace");
  const foreignWorkspace = newId("workspace");
  const projectId = newId("project");
  const otherProject = newId("project");
  const tenant = { workspaceId, projectId };
  const appRows = [
    { ...tenant, id: newId("app"), name: "Target", slug: "target" },
    { ...tenant, id: newId("app"), name: "Current caller", slug: "current" },
    { ...tenant, id: newId("app"), name: "Snapshot caller", slug: "snapshot" },
    { ...tenant, id: newId("app"), name: "Other app", slug: "other" },
    {
      ...tenant,
      workspaceId: foreignWorkspace,
      id: newId("app"),
      name: "Foreign caller",
      slug: "foreign",
    },
    {
      ...tenant,
      projectId: otherProject,
      id: newId("app"),
      name: "Other project caller",
      slug: "other-project",
    },
  ];
  const [targetApp, currentApp, snapshotApp, , foreignApp, otherProjectApp] = appRows;
  await database.insert(schema.apps).values(appRows);
  const environmentRows = appRows.map((app) => ({
    id: newId("environment"),
    workspaceId: app.workspaceId,
    projectId: app.projectId,
    appId: app.id,
    slug: "preview",
  }));
  const [targetEnv, currentEnv, snapshotEnv, wrongOwnerEnv, foreignEnv, otherProjectEnv] =
    environmentRows;
  await database.insert(schema.environments).values(environmentRows);
  const target = deploymentSeed({
    ...tenant,
    id: newId("test"),
    appId: targetApp.id,
    environmentId: targetEnv.id,
    status: "ready",
    capabilities: { private_networking: true },
    createdAt: 1,
  });
  // These mismatched tenant/owner rows exercise isolation against corrupt saved state.
  const foreignTarget = {
    ...target,
    id: newId("test"),
    k8sName: newId("test"),
    workspaceId: foreignWorkspace,
  };
  const otherProjectTarget = {
    ...target,
    id: newId("test"),
    k8sName: newId("test"),
    projectId: otherProject,
  };
  const ready = deploymentSeed({
    ...tenant,
    id: newId("test"),
    appId: snapshotApp.id,
    environmentId: snapshotEnv.id,
    status: "ready",
    capabilities: { private_networking: true },
    createdAt: 10,
  });
  const snapshotCases: { name: string; deployment: typeof schema.deployments.$inferInsert }[] = [
    { name: "Snapshot pin", deployment: ready },
    {
      name: "Awaiting pin",
      deployment: {
        ...ready,
        id: newId("test"),
        k8sName: newId("test"),
        status: "awaiting_approval",
        createdAt: 11,
      },
    },
    {
      name: "Stopped pin",
      deployment: {
        ...ready,
        id: newId("test"),
        k8sName: newId("test"),
        status: "stopped",
        createdAt: 12,
      },
    },
    {
      name: "Failed pin",
      deployment: {
        ...ready,
        id: newId("test"),
        k8sName: newId("test"),
        status: "failed",
        createdAt: 13,
      },
    },
    {
      name: "Desired stopped pin",
      deployment: {
        ...ready,
        id: newId("test"),
        k8sName: newId("test"),
        desiredState: "stopped",
        createdAt: 14,
      },
    },
    {
      name: "No capability pin",
      deployment: {
        ...ready,
        id: newId("test"),
        k8sName: newId("test"),
        capabilities: {},
        createdAt: 15,
      },
    },
    {
      name: "Wrong environment",
      deployment: {
        ...ready,
        id: newId("test"),
        k8sName: newId("test"),
        environmentId: wrongOwnerEnv.id,
        createdAt: 16,
      },
    },
    {
      name: "Foreign snapshot",
      deployment: {
        ...ready,
        id: newId("test"),
        k8sName: newId("test"),
        workspaceId: foreignWorkspace,
        appId: foreignApp.id,
        environmentId: foreignEnv.id,
        createdAt: 17,
      },
    },
    {
      name: "Other project snapshot",
      deployment: {
        ...ready,
        id: newId("test"),
        k8sName: newId("test"),
        projectId: otherProject,
        appId: otherProjectApp.id,
        environmentId: otherProjectEnv.id,
        createdAt: 18,
      },
    },
  ];
  await database
    .insert(schema.deployments)
    .values([
      target,
      foreignTarget,
      otherProjectTarget,
      ...snapshotCases.map(({ deployment }) => deployment),
    ]);
  const extraEnvironments = Array.from({ length: 2 }, (_, i) => ({
    ...currentEnv,
    id: newId("environment"),
    slug: `current-${i}`,
  }));
  await database.insert(schema.environments).values(extraEnvironments);
  const current = {
    ...tenant,
    id: newId("connection"),
    appId: currentApp.id,
    environmentId: currentEnv.id,
    resourceType: "app",
    resourceId: targetApp.id,
    name: "Current pin",
  };
  const retainedCurrent = {
    ...current,
    id: newId("connection"),
    environmentId: extraEnvironments[1].id,
    name: "Retained current",
  };
  const currentRules = [
    current,
    {
      ...current,
      id: newId("connection"),
      environmentId: extraEnvironments[0].id,
      name: "Automatic current",
    },
    retainedCurrent,
    {
      ...current,
      id: newId("connection"),
      workspaceId: foreignWorkspace,
      appId: foreignApp.id,
      environmentId: foreignEnv.id,
      name: "Foreign current",
    },
    {
      ...current,
      id: newId("connection"),
      projectId: otherProject,
      appId: otherProjectApp.id,
      environmentId: otherProjectEnv.id,
      name: "Other project current",
    },
  ];
  await database.insert(schema.appConnections).values(currentRules);
  // Stale target columns must not make automatic rules count as deployment pins.
  await database.insert(schema.connectionAppTargets).values(
    currentRules.map((rule) => ({
      connectionId: rule.id,
      selectionMode:
        rule.name === "Automatic current" ? ("automatic" as const) : ("deployment" as const),
      targetDeploymentId: target.id,
    })),
  );
  const snapshots = snapshotCases.map(({ name, deployment }) => ({
    workspaceId: deployment.workspaceId,
    projectId: deployment.projectId,
    appId: deployment.appId,
    environmentId: deployment.environmentId,
    deploymentId: deployment.id,
    connectionId: newId("connection"),
    resourceType: "app",
    resourceId: targetApp.id,
    name,
    createdAt: deployment.createdAt ?? 10,
  }));
  const retainedSnapshot = {
    ...snapshots[0],
    connectionId: newId("connection"),
    name: "Retained snapshot",
  };
  const allSnapshots = [
    ...snapshots,
    retainedSnapshot,
    { ...snapshots[0], connectionId: newId("connection"), name: "Automatic snapshot" },
  ];
  await database.insert(schema.deploymentConnections).values(allSnapshots);
  await database.insert(schema.deploymentConnectionAppTargets).values(
    allSnapshots.map((snapshot) => ({
      deploymentId: snapshot.deploymentId,
      connectionId: snapshot.connectionId,
      selectionMode:
        snapshot.name === "Automatic snapshot" ? ("automatic" as const) : ("deployment" as const),
      targetDeploymentId: target.id,
    })),
  );
  // A target row for another deployment must not fill a missing composite-key match.
  await database.insert(schema.deploymentConnectionAppTargets).values({
    deploymentId: newId("test"),
    connectionId: snapshots[0].connectionId,
    selectionMode: "deployment",
    targetDeploymentId: newId("test"),
  });
  await database.insert(schema.deploymentConnections).values({
    ...snapshots[0],
    connectionId: snapshots[2].connectionId,
    name: "Missing target settings",
  });

  const t = initTRPC.context<{ workspace: { id: string } }>().create();
  const caller = t
    .router({ list: listDeploymentConnectionPins })
    .createCaller({ workspace: { id: workspaceId } });
  expect(await caller.list({ deploymentId: target.id, projectId })).toEqual({
    pins: [
      {
        connectionId: current.id,
        connectionName: "Current pin",
        callerAppId: currentApp.id,
        callerAppName: "Current caller",
        provenance: "current_rule",
      },
      {
        connectionId: retainedCurrent.id,
        connectionName: "Retained current",
        callerAppId: currentApp.id,
        callerAppName: "Current caller",
        provenance: "current_rule",
      },
      {
        connectionId: retainedSnapshot.connectionId,
        connectionName: "Retained snapshot",
        callerAppId: snapshotApp.id,
        callerAppName: "Snapshot caller",
        callerDeploymentId: ready.id,
        callerStatus: "ready",
        provenance: "deployment_snapshot",
      },
      {
        connectionId: snapshots[0].connectionId,
        connectionName: "Snapshot pin",
        callerAppId: snapshotApp.id,
        callerAppName: "Snapshot caller",
        callerDeploymentId: ready.id,
        callerStatus: "ready",
        provenance: "deployment_snapshot",
      },
    ],
    truncated: false,
  });
  for (const deployment of [foreignTarget, otherProjectTarget]) {
    expect(await caller.list({ deploymentId: deployment.id, projectId })).toEqual({
      pins: [],
      truncated: false,
    });
  }
  const limitTarget = { ...target, id: newId("test"), k8sName: newId("test"), createdAt: 20 };
  await database.insert(schema.deployments).values(limitTarget);
  const limitEnvironments = Array.from({ length: 11 }, (_, i) => ({
    ...currentEnv,
    id: newId("environment"),
    slug: `limit-${i}`,
  }));
  await database.insert(schema.environments).values(limitEnvironments);
  const limitRules = limitEnvironments.map((environment, i) => ({
    ...current,
    id: newId("connection"),
    environmentId: environment.id,
    name: `Limit ${i.toString().padStart(2, "0")}`,
  }));
  await database.insert(schema.appConnections).values(limitRules);
  await database.insert(schema.connectionAppTargets).values(
    limitRules.map((rule) => ({
      connectionId: rule.id,
      selectionMode: "deployment" as const,
      targetDeploymentId: limitTarget.id,
    })),
  );
  const limited = await caller.list({ deploymentId: limitTarget.id, projectId });
  expect(limited.truncated).toBe(true);
  expect(limited.pins).toHaveLength(10);
  expect(limited.pins.map(({ connectionName }) => connectionName)).toEqual(
    Array.from({ length: 10 }, (_, index) => `Limit ${index.toString().padStart(2, "0")}`),
  );
});
