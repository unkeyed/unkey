import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type { V1KeysWhoAmIRequest, V1KeysWhoAmIResponse } from "./v1_keys_whoami";

runCommonRouteTests<V1KeysWhoAmIRequest>({
  prepareRequest: async (h) => {
    const { key } = await h.createKey();
    return {
      method: "POST",
      url: "/v1/keys.whoami",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key: key,
      },
    };
  },
});

describe("correct permissions", () => {
  describe.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
    { name: "wildcard api", roles: ["api.*.read_key", "api.*.read_api"] },
    {
      name: "wildcard mixed",
      roles: ["api.*.read_key", (apiId: string) => `api.${apiId}.read_api`],
    },
    {
      name: "wildcard mixed 2",
      roles: ["api.*.read_api", (apiId: string) => `api.${apiId}.read_key`],
    },
    { name: "wildcard and more", roles: ["api.*.read_key", "api.*.read_api", randomUUID()] },
    {
      name: "specific apiId",
      roles: [
        (apiId: string) => `api.${apiId}.read_key`,
        (apiId: string) => `api.${apiId}.read_api`,
      ],
    },
    {
      name: "specific apiId and more",
      roles: [
        (apiId: string) => `api.${apiId}.read_key`,
        (apiId: string) => `api.${apiId}.read_api`,
        randomUUID(),
      ],
    },
  ])("$name", ({ roles }) => {
    test("returns 200", async (t) => {
      const h = await IntegrationHarness.init(t);
      const { key } = await h.createKey();

      const root = await h.createRootKey(
        roles.map((role) => (typeof role === "string" ? role : role(h.resources.userApi.id))),
      );

      const res = await h.post<V1KeysWhoAmIRequest, V1KeysWhoAmIResponse>({
        url: "/v1/keys.whoami",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          key: key,
        },
      });

      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toEqual(200);
    });
  });
});
