import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { describe, expect, test } from "vitest";
import type {
  V1RatelimitSetOverrideRequest,
  V1RatelimitSetOverrideResponse,
} from "./v1_ratelimits_setOverride";

runCommonRouteTests<V1RatelimitSetOverrideRequest>({
  prepareRequest: async (rh) => {
    const identifier = randomUUID();
    const namespaceId = newId("test");

    const namespace = {
      id: namespaceId,
      workspaceId: rh.resources.userWorkspace.id,
      name: randomUUID(),
      createdAtM: Date.now(),
    };

    await rh.db.primary.insert(schema.ratelimitNamespaces).values(namespace);

    await rh.db.primary.query.ratelimitNamespaces.findFirst({
      where: (table, { eq }) => eq(table.id, namespaceId),
    });

    const override = {
      namespaceId: namespaceId,
      identifier: identifier,
      limit: 10,
      duration: 6500,
      async: true,
    };
    return {
      method: "POST",
      url: "/v1/ratelimits.setOverride",
      headers: {
        "Content-Type": "application/json",
      },
      body: override,
    };
  },
});
describe("correct roles", () => {
  describe.each([{ name: "set override", roles: ["ratelimit.*.set_override"] }])(
    "$name",
    ({ roles }) => {
      test("returns 200", async (t) => {
        const h = await IntegrationHarness.init(t);
        const identifier = randomUUID();
        const namespaceId = newId("test");
        const namespace = {
          id: namespaceId,
          workspaceId: h.resources.userWorkspace.id,
          createdAtM: Date.now(),
          name: newId("test"),
        };
        await h.db.primary.insert(schema.ratelimitNamespaces).values(namespace);

        const root = await h.createRootKey(roles);

        const res = await h.post<V1RatelimitSetOverrideRequest, V1RatelimitSetOverrideResponse>({
          url: "/v1/ratelimits.setOverride",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${root.key}`,
          },
          body: {
            namespaceId,
            identifier,
            limit: 1,
            duration: 60_000,
            async: false,
          },
        });

        expect(
          res.status,
          `expected status 200, received: ${JSON.stringify(res, null, 2)}`,
        ).toEqual(200);
        await h.post<V1RatelimitSetOverrideRequest, V1RatelimitSetOverrideResponse>({
          url: "/v1/ratelimits.setOverride",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${root.key}`,
          },
          body: {
            namespaceId,
            identifier,
            limit: 7,
            duration: 60_000,
            async: false,
          },
        });
        expect(
          res.status,
          `expected status 200, received: ${JSON.stringify(res, null, 2)}`,
        ).toEqual(200);

        const found = await h.db.primary.query.ratelimitOverrides.findFirst({
          where: (table, { eq }) => eq(table.id, res.body.overrideId),
        });

        expect(found?.limit).toEqual(7);
      });
    },
  );
});

describe("incorrect roles", () => {
  describe.each([
    { name: "empty roles", roles: [] },
    { name: "invalid role", roles: ["wrong.role"] },
    { name: "insufficient role", roles: ["ratelimit.*.view"] },
  ])("$name", ({ roles }) => {
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

      const root = await h.createRootKey(roles);

      const res = await h.post<V1RatelimitSetOverrideRequest, V1RatelimitSetOverrideResponse>({
        url: "/v1/ratelimits.setOverride",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          namespaceId,
          identifier,
          limit: 1,
          duration: 60_000,
          async: false,
        },
      });

      expect(res.status, `expected status 403, received: ${JSON.stringify(res, null, 2)}`).toEqual(
        403,
      );

      const found = await h.db.primary.query.ratelimitOverrides.findFirst({
        where: (table, { eq }) => eq(table.id, overrideId),
      });
      expect(found?.id).toEqual(undefined);
    });
  });
});
