import { initTRPC } from "@trpc/server";
import { asc, eq, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { expect, it, vi } from "vitest";
import {
  connectionTestDatabase,
  deploymentSeed,
} from "../../../../../../../internal/db/src/testing";
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
    const database = await connectionTestDatabase(databaseUrl);
    queries.select.mockImplementation(database.select.bind(database));
    queries.findApp.mockImplementation(database.query.apps.findFirst.bind(database.query.apps));
    const workspaceId = newId("workspace");
    const projectId = newId("project");
    const web = newId("app");
    const api = newId("app");
    const foreign = newId("app");
    const preview = newId("environment");
    const apiPreview = newId("environment");
    const olderLivePin = newId("test");
    const tenant = { workspaceId, projectId };
    await database.insert(schema.apps).values([
      { ...tenant, id: web, name: "Web", slug: "web" },
      { ...tenant, id: api, name: "API", slug: "api" },
      { ...tenant, id: foreign, workspaceId: newId("workspace"), name: "Foreign", slug: "foreign" },
    ]);
    await database.insert(schema.environments).values([
      { ...tenant, id: preview, appId: web, slug: "preview" },
      { ...tenant, id: apiPreview, appId: api, slug: "preview" },
    ]);
    const otherEnvironments = Array.from({ length: 501 }, (_, i) => ({
      ...tenant,
      id: newId("environment"),
      appId: web,
      slug: `other-${i}`,
    }));
    await database.insert(schema.environments).values(otherEnvironments);
    const otherRules = otherEnvironments.map((environment, i) => ({
      ...tenant,
      id: newId("connection"),
      appId: web,
      environmentId: environment.id,
      resourceType: "app",
      resourceId: api,
      name: `a-other-${i}`,
    }));
    // Dangling, self, and cross-tenant targets deliberately exercise corrupt saved rules.
    const invalidRules = [web, foreign, ...Array.from({ length: 501 }, () => newId("app"))].map(
      (id, i) => ({
        ...tenant,
        id: newId("connection"),
        appId: web,
        environmentId: preview,
        resourceType: "app",
        resourceId: id,
        name: `b-invalid-${i}`,
      }),
    );
    const selected = {
      ...tenant,
      id: newId("connection"),
      appId: web,
      environmentId: preview,
      resourceType: "app",
      resourceId: api,
      name: "z-selected",
    };
    const rules = [...otherRules, ...invalidRules, selected];
    await database.insert(schema.appConnections).values(rules);
    const otherPin = newId("test");
    await database.insert(schema.connectionAppTargets).values(
      rules.map(({ id }) => ({
        connectionId: id,
        selectionMode: "deployment" as const,
        targetDeploymentId: id === selected.id ? olderLivePin : otherPin,
      })),
    );
    const newerIds = Array.from({ length: 101 }, () => newId("test"));
    await database.insert(schema.deployments).values([
      deploymentSeed({
        ...tenant,
        id: olderLivePin,
        appId: api,
        environmentId: apiPreview,
        status: "ready",
        createdAt: 1,
        firstReadyAt: 1,
      }),
      ...newerIds.map((id) =>
        deploymentSeed({
          ...tenant,
          id,
          appId: api,
          environmentId: apiPreview,
          status: "ready",
          createdAt: 100,
          firstReadyAt: 100,
        }),
      ),
    ]);

    const newer = await database
      .select({ id: schema.deployments.id })
      .from(schema.deployments)
      .where(eq(schema.deployments.createdAt, 100))
      .orderBy(asc(schema.deployments.id));
    const t = initTRPC.context<{ workspace: { id: string } }>().create();
    const caller = t
      .router({ targets: listAppConnectionTargets })
      .createCaller({ workspace: { id: workspaceId } });
    const input = { projectId, appId: web, environmentId: preview, targetAppId: api };
    const first = await caller.targets(input);
    expect(first.deployments).toHaveLength(101);
    expect(first.deployments[0]).toMatchObject({ id: olderLivePin, status: "ready" });
    expect(first.deployments.slice(1).map(({ id }) => id)).toEqual(
      newer
        .slice(1)
        .reverse()
        .map(({ id }) => id),
    );
    expect(first.nextCursor).toEqual({ id: newer[1].id, createdAt: 100 });
    const second = await caller.targets({ ...input, cursor: first.nextCursor });
    expect(second.deployments.map(({ id }) => id)).toEqual([olderLivePin, newer[0].id]);
    expect(second.nextCursor).toBeNull();
    const other = await caller.targets({
      ...input,
      environmentId: otherEnvironments[0].id,
      targetAppId: undefined,
    });
    expect(other.deployments).toEqual([]);
  },
);
