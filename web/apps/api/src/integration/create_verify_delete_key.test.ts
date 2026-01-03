import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import type { V1ApisCreateApiRequest, V1ApisCreateApiResponse } from "@/routes/v1_apis_createApi";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "@/routes/v1_keys_createKey";
import type { V1KeysDeleteKeyRequest, V1KeysDeleteKeyResponse } from "@/routes/v1_keys_deleteKey";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "@/routes/v1_keys_verifyKey";
import { expect, test } from "vitest";

test("create, verify and delete a key", async (t) => {
  const h = await IntegrationHarness.init(t);
  const { key: rootKey } = await h.createRootKey(["*"]);

  const createApiResponse = await h.post<V1ApisCreateApiRequest, V1ApisCreateApiResponse>({
    url: `${h.baseUrl}/v1/apis.createApi`,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      name: "scenario-test-pls-delete",
    },
  });
  expect(createApiResponse.status).toEqual(200);
  expect(createApiResponse.body.apiId).toBeDefined();
  expect(createApiResponse.headers).toHaveProperty("unkey-request-id");

  const key = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: `${h.baseUrl}/v1/keys.createKey`,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      apiId: createApiResponse.body.apiId,
      byteLength: 16,
      enabled: true,
    },
  });
  expect(key.status).toEqual(200);
  expect(key.body.key).toBeDefined();
  expect(key.body.keyId).toBeDefined();

  const valid = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: `${h.baseUrl}/v1/keys.verifyKey`,
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      key: key.body.key,
      apiId: createApiResponse.body.apiId,
    },
  });
  expect(valid.status).toEqual(200);
  expect(valid.body.valid).toEqual(true);

  const revoked = await h.post<V1KeysDeleteKeyRequest, V1KeysDeleteKeyResponse>({
    url: `${h.baseUrl}/v1/keys.deleteKey`,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      keyId: key.body.keyId,
    },
  });
  expect(revoked.status).toEqual(200);

  const validAfterRevoke = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: `${h.baseUrl}/v1/keys.verifyKey`,
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      key: key.body.key,
      apiId: createApiResponse.body.apiId,
    },
  });
  expect(validAfterRevoke.status).toEqual(200);
  expect(validAfterRevoke.body.valid).toEqual(false);
});
