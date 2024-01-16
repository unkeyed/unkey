import { integrationTestEnv } from "@/pkg/testutil/env";
import { step } from "@/pkg/testutil/request";
import type { V1ApisCreateApiRequest, V1ApisCreateApiResponse } from "@/routes/v1_apis_createApi";
import type { V1ApisDeleteApiRequest, V1ApisDeleteApiResponse } from "@/routes/v1_apis_deleteApi";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "@/routes/v1_keys_createKey";
import { V1KeysGetKeyResponse } from "@/routes/v1_keys_getKey";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "@/routes/v1_keys_verifyKey";
import { expect, test } from "vitest";

const env = integrationTestEnv.parse(process.env);

test("remaining consistently counts down", async () => {
  const createApiResponse = await step<V1ApisCreateApiRequest, V1ApisCreateApiResponse>({
    url: `${env.UNKEY_BASE_URL}/v1/apis.createApi`,
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${env.UNKEY_ROOT_KEY}`,
    },
    body: {
      name: "scenario-test-pls-delete",
    },
  });
  expect(createApiResponse.status).toEqual(200);
  expect(createApiResponse.body.apiId).toBeDefined();
  expect(createApiResponse.headers).toHaveProperty("unkey-request-id");

  const remaining = 100;

  const createKeyResponse = await step<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: `${env.UNKEY_BASE_URL}/v1/keys.createKey`,
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${env.UNKEY_ROOT_KEY}`,
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
    const valid = await step<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: `${env.UNKEY_BASE_URL}/v1/keys.verifyKey`,
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${env.UNKEY_ROOT_KEY}`,
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

  const invalid = await step<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: `${env.UNKEY_BASE_URL}/v1/keys.verifyKey`,
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${env.UNKEY_ROOT_KEY}`,
    },
    body: {
      apiId: createApiResponse.body.apiId,
      key: createKeyResponse.body.key,
    },
  });
  expect(invalid.status).toEqual(200);
  expect(invalid.body.valid).toBe(false);
  expect(invalid.body.remaining).toEqual(0);

  const key = await step<never, V1KeysGetKeyResponse>({
    url: `${env.UNKEY_BASE_URL}/v1/keys.getKey?keyId=${createKeyResponse.body.keyId}`,
    method: "GET",
    headers: {
      Authorization: `Bearer ${env.UNKEY_ROOT_KEY}`,
    },
  });
  expect(key.status).toEqual(200);
  expect(key.body.id).toEqual(createKeyResponse.body.keyId);
  expect(key.body.remaining).toBeDefined();
  expect(key.body.remaining).toEqual(0);

  /**
   * Teardown
   */
  const deleteApi = await step<V1ApisDeleteApiRequest, V1ApisDeleteApiResponse>({
    url: `${env.UNKEY_BASE_URL}/v1/apis.deleteApi`,
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${env.UNKEY_ROOT_KEY}`,
    },
    body: {
      apiId: createApiResponse.body.apiId,
    },
  });
  expect(deleteApi.status).toEqual(200);
}, 60_000);
