import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type { V1KeysSetRolesRequest, V1KeysSetRolesResponse } from "./v1_keys_setRoles";

runCommonRouteTests<V1KeysSetRolesRequest>({
  prepareRequest: async (h) => {
    const { keyId } = await h.createKey();

    return {
      method: "POST",
      url: "/v1/keys.setRoles",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        keyId,
        roles: [
          {
            name: "hello",
            create: true,
          },
        ],
      },
    };
  },
});
describe("correct permissions", () => {
  describe.each([
    { name: "legacy", permissions: ["*"] },
    { name: "legacy and more", permissions: ["*", randomUUID()] },
  ])("$name", ({ permissions }) => {
    test("returns 200", async (t) => {
      const h = await IntegrationHarness.init(t);
      const root = await h.createRootKey(permissions);

      const { keyId } = await h.createKey();

      const res = await h.post<V1KeysSetRolesRequest, V1KeysSetRolesResponse>({
        url: "/v1/keys.setRoles",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          keyId,
          roles: [
            {
              name: "hello",
              create: true,
            },
            { name: "there", create: true },
          ],
        },
      });

      expect(res.status, `expected status 200, received: ${JSON.stringify(res)}`).toEqual(200);

      const found = await h.db.readonly.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, keyId),
        with: {
          roles: {
            with: {
              role: true,
            },
          },
        },
      });
      expect(found).toBeDefined();
      expect(found!.roles.length).toBe(2);
    });
  });
});
