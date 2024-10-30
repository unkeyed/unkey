import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import type { V1ApisCreateApiRequest, V1ApisCreateApiResponse } from "@/routes/v1_apis_createApi";
import type { V1ApisDeleteApiRequest, V1ApisDeleteApiResponse } from "@/routes/v1_apis_deleteApi";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "@/routes/v1_keys_createKey";
import type { V1KeysGetKeyResponse } from "@/routes/v1_keys_getKey";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "@/routes/v1_keys_verifyKey";
import { expect, test } from "vitest";

test("remaining consistently counts down", async (t) => {
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

  const remaining = 100;

  const createKeyResponse = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: `${h.baseUrl}/v1/keys.createKey`,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      apiId: createApiResponse.body.apiId,
      byteLength: 32,
      prefix: "test",
      remaining,
      enabled: true,
    },
  });
  expect(createKeyResponse.status).toEqual(200);

  for (let i = remaining - 1; i >= 0; i--) {
    const valid = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: `${h.baseUrl}/v1/keys.verifyKey`,
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        apiId: createApiResponse.body.apiId,
        key: createKeyResponse.body.key,
      },
    });

    expect(valid.status).toEqual(200);
    expect(valid.body.valid).toBe(true);
    expect(valid.body.remaining).toEqual(i);
  }

  const invalid = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: `${h.baseUrl}/v1/keys.verifyKey`,
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      apiId: createApiResponse.body.apiId,
      key: createKeyResponse.body.key,
    },
  });
  expect(invalid.status).toEqual(200);
  expect(invalid.body.valid).toBe(false);
  expect(invalid.body.remaining).toEqual(0);

  // wait until the updates can propagate from the durable object to the db
  await new Promise((r) => setTimeout(r, 2000));

  const key = await h.get<V1KeysGetKeyResponse>({
    url: `${h.baseUrl}/v1/keys.getKey?keyId=${createKeyResponse.body.keyId}`,

    headers: {
      Authorization: `Bearer ${rootKey}`,
    },
  });
  expect(key.status).toEqual(200);
  expect(key.body.id).toEqual(createKeyResponse.body.keyId);
  expect(key.body.remaining).toBeDefined();
  expect(key.body.remaining).toEqual(0);

  /**
   * Teardown
   */
  const deleteApi = await h.post<V1ApisDeleteApiRequest, V1ApisDeleteApiResponse>({
    url: `${h.baseUrl}/v1/apis.deleteApi`,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      apiId: createApiResponse.body.apiId,
    },
  });
  expect(deleteApi.status).toEqual(200);
}, 60_000);
