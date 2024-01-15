import { integrationTestEnv } from "@/pkg/testutil/env";
import { step } from "@/pkg/testutil/request";
import type { V1ApisCreateApiRequest, V1ApisCreateApiResponse } from "@/routes/v1_apis_createApi";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "@/routes/v1_keys_createKey";
import { V1KeysDeleteKeyRequest, V1KeysDeleteKeyResponse } from "@/routes/v1_keys_deleteKey";
import { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "@/routes/v1_keys_verifyKey";
import { afterAll, expect, test } from "vitest";

const apiIds: string[] = [];

const env = integrationTestEnv.parse(process.env);
test("create, verify and delete a key", async () => {
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
  apiIds.push(createApiResponse.body.apiId);
  expect(createApiResponse.headers).toHaveProperty("unkey-request-id");

  const key = await step<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: `${env.UNKEY_BASE_URL}/v1/keys.createKey`,
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${env.UNKEY_ROOT_KEY}`,
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

  const valid = await step<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: `${env.UNKEY_BASE_URL}/v1/keys.verifyKey`,
    method: "POST",
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

  const revoked = await step<V1KeysDeleteKeyRequest, V1KeysDeleteKeyResponse>({
    url: `${env.UNKEY_BASE_URL}/v1/keys.deleteKey`,
    method: "POST",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${env.UNKEY_ROOT_KEY}`,
    },
    body: {
      keyId: key.body.keyId,
    },
  });
  expect(revoked.status).toEqual(200);

  const validAfterRevoke = await step<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: `${env.UNKEY_BASE_URL}/v1/keys.verifyKey`,
    method: "POST",
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

afterAll(async () => {
  for (const apiId of apiIds) {
    await step({
      url: `${env.UNKEY_BASE_URL}/v1/apis.deleteApi`,
      method: "POST",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${env.UNKEY_ROOT_KEY}`,
      },
      body: {
        apiId,
      },
    });
  }
});
