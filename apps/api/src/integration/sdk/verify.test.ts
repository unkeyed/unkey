import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "@/routes/v1_keys_verifyKey";
import { expect, test } from "vitest";

test("without permissions", async () => {
  using h = new IntegrationHarness();
  await h.seed();

  const { key } = await h.createKey();

  const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: `${h.baseUrl}/v1/keys.verifyKey`,
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      apiId: h.resources.userApi.id,
      key,
      authorization: {
        permissions: {
          version: 1,
          query: {
            and: ["p1", "p2"],
          },
        },
      },
    },
  });

  expect(res.status).toBe(200);
  expect(res.body.valid).toBe(false);
  expect(res.body.code).toBe("INSUFFICIENT_PERMISSIONS");
});
