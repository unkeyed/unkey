import { test, expect, afterAll } from "bun:test";
import { step } from "@/pkg/testutil/step";
import { testEnv } from "./env";
import type { V1ApisCreateApiRequest, V1ApisCreateApiResponse } from "@/routes/v1_apis_createApi";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "@/routes/v1_keys_createKey";
import type { V1ApisListKeysResponse } from "@/routes/v1_apis_listKeys";
import { V1ApisDeleteApiRequest, V1ApisDeleteApiResponse } from "@/routes/v1_apis_deleteApi";

const env = testEnv();
test("update a key's remaining limit", async () => {
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

  afterAll(async () => {
    await step<V1ApisDeleteApiRequest, V1ApisDeleteApiResponse>({
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
  });

  for (let i = 0; i < 5; i++) {
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
      },
    });
    expect(createKeyResponse.status).toEqual(200);
    afterAll(async () => {
      await step({
        url: `${env.UNKEY_BASE_URL}/v1/keys.deleteKey`,
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${env.UNKEY_ROOT_KEY}`,
        },
        body: {
          keyId: createKeyResponse.body.keyId,
        },
      });
    });
  }
  const listKeysResponse = await step<never, V1ApisListKeysResponse>({
    url: `${env.UNKEY_BASE_URL}/v1/apis.listKeys?apiId=${createApiResponse.body.apiId}`,
    method: "GET",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${env.UNKEY_ROOT_KEY}`,
    },
  });

  expect(listKeysResponse.status).toEqual(200);
  expect(listKeysResponse.body.keys).toHaveLength(5);
});
