import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type {
  V1IdentitiesCreateIdentityRequest,
  V1IdentitiesCreateIdentityResponse,
} from "./v1_identities_createIdentity";

runCommonRouteTests<V1IdentitiesCreateIdentityRequest>({
  prepareRequest: () => ({
    method: "POST",
    url: "/v1/identities.createIdentity",
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      externalId: randomUUID(),
    },
  }),
});
describe("correct roles", () => {
  describe.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
    { name: "wildcard", roles: ["identity.*.create_identity"] },
    { name: "wildcard and more", roles: ["identity.*.create_identity", randomUUID()] },
  ])("$name", ({ roles }) => {
    test("returns 200", async (t) => {
      const h = await IntegrationHarness.init(t);
      const root = await h.createRootKey(roles);

      const res = await h.post<
        V1IdentitiesCreateIdentityRequest,
        V1IdentitiesCreateIdentityResponse
      >({
        url: "/v1/identities.createIdentity",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          externalId: randomUUID(),
        },
      });
      expect(res.status, `expected status 200, received: ${JSON.stringify(res, null, 2)}`).toEqual(
        200,
      );

      const found = await h.db.primary.query.identities.findFirst({
        where: (table, { eq }) => eq(table.id, res.body.identityId),
      });
      expect(found).toBeDefined();
    });
  });
});
