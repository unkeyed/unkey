import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type { V1IdentitiesListIdentitiesResponse } from "./v1_identities_listIdentities";

runCommonRouteTests({
  prepareRequest: async (rh) => {
    await rh.db.primary.insert(schema.identities).values({
      id: newId("test"),
      externalId: randomUUID(),
      workspaceId: rh.resources.userWorkspace.id,
    });
    return {
      method: "GET",
      url: "/v1/identities.listIdentities",
    };
  },
});

describe("correct permissions", () => {
  describe.each([
    { name: "legacy", permissions: ["*"] },
    { name: "legacy and more", permissions: ["*", randomUUID()] },
    { name: "wildcard api", permissions: ["identity.*.read_identity"] },
    {
      name: "wildcard mixed",
      permissions: ["identity.*.read_identity", (envId: string) => `identity.${envId}.identity`],
    },
    {
      name: "wildcard mixed 2",
      permissions: [
        "identity.*.read_identity",
        (envId: string) => `identity.${envId}.read_identity`,
      ],
    },
    { name: "wildcard and more", permissions: ["identity.*.read_identity", randomUUID()] },
    {
      name: "specific envId",
      permissions: [(envId: string) => `identity.${envId}.read_identity`],
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
        permissions.map((p) => (typeof p === "string" ? p : p("default"))),
      );

      const res = await h.get<V1IdentitiesListIdentitiesResponse>({
        url: "/v1/identities.listIdentities",
        headers: {
          Authorization: `Bearer ${root.key}`,
        },
      });
      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    });
  });
});
