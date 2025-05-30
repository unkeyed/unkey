import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { describe, expect, test } from "vitest";
import type { V1RatelimitGetOverrideResponse } from "./v1_ratelimits_getOverride";
import type {
  V1RatelimitListOverridesRequest,
  V1RatelimitListOverridesResponse,
} from "./v1_ratelimits_listOverrides";

runCommonRouteTests<V1RatelimitListOverridesRequest>({
  prepareRequest: async (rh) => {
    const overrideId = newId("test");
    const identifier = randomUUID();
    const namespaceId = newId("test");
    const namespace = {
      id: namespaceId,
      workspaceId: rh.resources.userWorkspace.id,
      createdAtM: Date.now(),
      name: randomUUID(),
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
      url: `/v1/ratelimits.listOverrides?namespaceId=${namespaceId}`,
      headers: {
        "Content-Type": "application/json",
      },
    };
  },
});
describe("correct roles", () => {
  describe.each([{ name: "list override", roles: ["ratelimit.*.read_override"] }])(
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
        const res = await h.get<V1RatelimitListOverridesResponse>({
          url: `/v1/ratelimits.listOverrides?namespaceId=${namespaceId}`,
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
  describe.each([
    { name: "no roles", roles: [] },
    { name: "insufficient roles", roles: ["ratelimit.*.write_override"] },
    { name: "wrong namespace permissions", roles: ["ratelimit.othernamespace.read_override"] },
    { name: "expired token", roles: ["ratelimit.*.read_override"], tokenExpired: true },
    { name: "invalid token", roles: ["ratelimit.*.read_override"], tokenInvalid: true },
  ])("$name", ({ roles, tokenExpired, tokenInvalid }) => {
    test("returns appropriate status code", async (t) => {
      const h = await IntegrationHarness.init(t);
      const overrideId = newId("test");
      const identifier = randomUUID();
      const namespaceId = newId("test");

      // Insert namespace and override into the database
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

      // Create root key with specified roles and token conditions
      const rootOptions: any = { roles };
      if (tokenExpired) {
        rootOptions.expiresAt = new Date(Date.now() - 60 * 60 * 1000); // Set expiration in the past
      }
      if (tokenInvalid) {
        rootOptions.key = "invalid_key"; // Use an invalid key
      }
      const root = await h.createRootKey(rootOptions);

      // Make the API request
      const res = await h.get<V1RatelimitGetOverrideResponse>({
        url: `/v1/ratelimits.getOverride?namespaceId=${namespaceId}&identifier=${identifier}`,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
      });

      // Determine the expected status code based on the scenario
      let expectedStatus = 200;
      if (
        !roles.includes("ratelimit.*.read_override") ||
        tokenExpired ||
        tokenInvalid ||
        roles.includes("ratelimit.othernamespace.read_override")
      ) {
        expectedStatus = 403;
      }

      expect(res.status).toEqual(expectedStatus);
    });
  });
});
