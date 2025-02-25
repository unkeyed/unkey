import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { describe, expect, test } from "vitest";
import type {
  V1RatelimitDeleteOverrideRequest,
  V1RatelimitDeleteOverrideResponse,
} from "./v1_ratelimits_deleteOverride";

runCommonRouteTests<V1RatelimitDeleteOverrideRequest>({
  prepareRequest: async (rh) => {
    const overrideId = newId("test");
    const identifier = randomUUID();
    const namespaceId = newId("test");
    const namespace = {
      id: namespaceId,
      workspaceId: rh.resources.userWorkspace.id,
      createdAtM: Date.now(),
      name: newId("test"),
    };
    await rh.db.primary.insert(schema.ratelimitNamespaces).values(namespace);
    await rh.db.primary.insert(schema.ratelimitOverrides).values({
      id: overrideId,
      workspaceId: rh.resources.userWorkspace.id,
      namespaceId,
      identifier,
      limit: 1,
      duration: 60_000,
      async: false,
    });

    return {
      method: "POST",
      url: "/v1/ratelimits.deleteOverride",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        namespaceId,
        identifier,
      },
    };
  },
});
describe("correct roles", () => {
  describe.each([{ name: "delete override", roles: ["ratelimit.*.delete_override"] }])(
    "$name",
    ({ roles }) => {
      test("returns 200", async (t) => {
        const h = await IntegrationHarness.init(t);
        const overrideId = newId("test");
        const identifier = randomUUID();
        const namespaceId = newId("test");
        const namespace = {
          id: namespaceId,
          workspaceId: h.resources.userWorkspace.id,
          createdAtM: Date.now(),
          name: newId("test"),
        };
        await h.db.primary.insert(schema.ratelimitNamespaces).values(namespace);
        await h.db.primary.insert(schema.ratelimitOverrides).values({
          id: overrideId,
          workspaceId: h.resources.userWorkspace.id,
          namespaceId,
          identifier,
          limit: 1,
          duration: 60_000,
          async: false,
        });
        const root = await h.createRootKey(roles);

        const res = await h.post<
          V1RatelimitDeleteOverrideRequest,
          V1RatelimitDeleteOverrideResponse
        >({
          url: "/v1/ratelimits.deleteOverride",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${root.key}`,
          },
          body: {
            namespaceId,
            identifier,
          },
        });

        expect(
          res.status,
          `expected status 200, received: ${JSON.stringify(res, null, 2)}`,
        ).toEqual(200);

        const found = await h.db.primary.query.ratelimitOverrides.findFirst({
          where: (table, { eq, and, isNull }) =>
            and(isNull(table.deletedAtM), eq(table.id, overrideId)),
        });
        expect(found).toBeUndefined();
      });
    },
  );
});

describe("incorrect roles", () => {
  describe.each([{ name: "delete override", roles: ["ratelimit.*.create_override"] }])(
    "$name",
    ({ roles }) => {
      test("returns 403", async (t) => {
        const h = await IntegrationHarness.init(t);
        const overrideId = newId("test");
        const identifier = randomUUID();
        const namespaceId = newId("test");
        const namespace = {
          id: namespaceId,
          workspaceId: h.resources.userWorkspace.id,
          createdAtM: Date.now(),
          name: newId("test"),
        };
        await h.db.primary.insert(schema.ratelimitNamespaces).values(namespace);
        await h.db.primary.insert(schema.ratelimitOverrides).values({
          id: overrideId,
          workspaceId: h.resources.userWorkspace.id,
          namespaceId,
          identifier,
          limit: 1,
          duration: 60_000,
          async: false,
        });
        const root = await h.createRootKey(roles);

        const res = await h.post<
          V1RatelimitDeleteOverrideRequest,
          V1RatelimitDeleteOverrideResponse
        >({
          url: "/v1/ratelimits.deleteOverride",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${root.key}`,
          },
          body: {
            namespaceId,
            identifier,
          },
        });

        expect(
          res.status,
          `expected status 403, received: ${JSON.stringify(res, null, 2)}`,
        ).toEqual(403);

        const found = await h.db.primary.query.ratelimitOverrides.findFirst({
          where: (table, { eq }) => eq(table.id, overrideId),
        });
        expect(found?.id).toEqual(overrideId);
      });
    },
  );
});
