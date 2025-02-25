import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { describe, expect, test } from "vitest";
import type {
  V1RatelimitGetOverrideRequest,
  V1RatelimitGetOverrideResponse,
} from "./v1_ratelimits_getOverride";

runCommonRouteTests<V1RatelimitGetOverrideRequest>({
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
      method: "GET",
      url: `/v1/ratelimits.getOverride?namespaceId=${namespaceId}&identifier=${identifier}`,
      headers: {
        "Content-Type": "application/json",
      },
    };
  },
});
describe("correct roles", () => {
  describe.each([{ name: "get override", roles: ["ratelimit.*.read_override"] }])(
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
          name: randomUUID(),
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
        const res = await h.get<V1RatelimitGetOverrideResponse>({
          url: `/v1/ratelimits.getOverride?namespaceId=${namespaceId}&identifier=${identifier}`,
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${root.key}`,
          },
        });

        expect(
          res.status,
          `expected status 200, received: ${JSON.stringify(res, null, 2)}`,
        ).toEqual(200);
      });
    },
  );
});

describe("incorrect roles", () => {
  describe.each([{ name: "get override", roles: [] }])("$name", ({ roles }) => {
    test("returns 403", async (t) => {
      const h = await IntegrationHarness.init(t);
      const overrideId = newId("test");
      const identifier = randomUUID();
      const namespaceId = newId("test");
      const namespace = {
        id: namespaceId,
        workspaceId: h.resources.userWorkspace.id,
        createdAtM: Date.now(),
        name: randomUUID(),
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
      const res = await h.get<V1RatelimitGetOverrideResponse>({
        url: `/v1/ratelimits.getOverride?namespaceId=${namespaceId}&identifier=${identifier}`,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
      });
      expect(res.status, `expected status 403, received: ${JSON.stringify(res, null, 2)}`).toEqual(
        403,
      );
    });
  });
});
