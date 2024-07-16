import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type {
  V1IdentitiesDeleteIdentityRequest,
  V1IdentitiesDeleteIdentityResponse,
} from "./v1_identities_deleteIdentity";

runCommonRouteTests<V1IdentitiesDeleteIdentityRequest>({
  prepareRequest: async (rh) => {
    const identityId = newId("test");
    await rh.db.primary.insert(schema.identities).values({
      id: identityId,
      externalId: randomUUID(),
      workspaceId: rh.resources.userWorkspace.id,
    });
    return {
      method: "POST",
      url: "/v1/identities.deleteIdentity",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        identityId,
      },
    };
  },
});

describe("correct permissions", () => {
  describe.each([
    { name: "wildcard", permissions: ["identity.*.delete_identity"] },
    { name: "wildcard and more", permissions: ["identity.*.delete_identity", randomUUID()] },
    {
      name: "specific identityId",
      permissions: [(identityId: string) => `identity.${identityId}.delete_identity`],
    },
    {
      name: "specific identityId and more",
      permissions: [(identityId: string) => `identity.${identityId}.delete_identity`, randomUUID()],
    },
  ])("$name", ({ permissions }) => {
    test("returns 200", async (t) => {
      const h = await IntegrationHarness.init(t);
      const identityId = newId("test");
      await h.db.primary.insert(schema.identities).values({
        id: identityId,
        externalId: randomUUID(),
        workspaceId: h.resources.userWorkspace.id,
      });
      const root = await h.createRootKey(
        permissions.map((p) => (typeof p === "string" ? p : p(identityId))),
      );

      const res = await h.post<
        V1IdentitiesDeleteIdentityRequest,
        V1IdentitiesDeleteIdentityResponse
      >({
        url: "/v1/identities.deleteIdentity",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          identityId,
        },
      });
      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

      const found = await h.db.primary.query.identities.findFirst({
        where: (table, { eq }) => eq(table.id, identityId),
      });
      expect(found).toBeUndefined();
    });
  });
});
