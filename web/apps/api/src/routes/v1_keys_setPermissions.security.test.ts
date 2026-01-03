import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type {
  V1KeysSetPermissionsRequest,
  V1KeysSetPermissionsResponse,
} from "./v1_keys_setPermissions";

runCommonRouteTests<V1KeysSetPermissionsRequest>({
  prepareRequest: async (h) => {
    const { keyId } = await h.createKey();

    return {
      method: "POST",
      url: "/v1/keys.setPermissions",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        keyId,
        permissions: [
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

      const res = await h.post<V1KeysSetPermissionsRequest, V1KeysSetPermissionsResponse>({
        url: "/v1/keys.setPermissions",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          keyId,
          permissions: [
            {
              name: "hello",
              create: true,
            },
            { name: "there", create: true },
          ],
        },
      });

      expect(res.status, `expected status 200, received: ${JSON.stringify(res, null, 2)}`).toEqual(
        200,
      );

      const found = await h.db.primary.query.keys.findFirst({
        where: (table, { eq }) => eq(table.id, keyId),
        with: {
          permissions: {
            with: {
              permission: true,
            },
          },
        },
      });
      expect(found).toBeDefined();
      expect(found!.permissions.length).toBe(2);
    });
  });
});
