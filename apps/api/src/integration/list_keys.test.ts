import { integrationTestEnv } from "@/pkg/testutil/env";
import { step } from "@/pkg/testutil/request";
import type { V1ApisCreateApiRequest, V1ApisCreateApiResponse } from "@/routes/v1_apis_createApi";
import { V1ApisDeleteApiRequest, V1ApisDeleteApiResponse } from "@/routes/v1_apis_deleteApi";
import type { V1ApisListKeysResponse } from "@/routes/v1_apis_listKeys";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "@/routes/v1_keys_createKey";
import { V1KeysDeleteKeyRequest } from "@/routes/v1_keys_deleteKey";
import { expect, test } from "vitest";

const env = integrationTestEnv.parse(process.env);
test("create and list keys", async () => {
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
        enabled: true,
      },
    });
    expect(createKeyResponse.status).toEqual(200);
    expect(createKeyResponse.body.keyId).toBeDefined();
    expect(createKeyResponse.body.key).toBeDefined();
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
});

test("list keys does not return revoked keys", async () => {
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

  const activeKeyIds: string[] = [];
  const deletedKeyIds: string[] = [];

  for (let i = 0; i < 10; i++) {
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
        enabled: true,
      },
    });
    expect(createKeyResponse.status).toEqual(200);
    expect(createKeyResponse.body.keyId).toBeDefined();
    expect(createKeyResponse.body.key).toBeDefined();

    if (i % 2 === 0) {
      const deleteKeyResponse = await step<V1KeysDeleteKeyRequest, V1KeysDeleteKeyRequest>({
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
      expect(deleteKeyResponse.status).toEqual(200);
      deletedKeyIds.push(createKeyResponse.body.keyId);
    } else {
      activeKeyIds.push(createKeyResponse.body.keyId);
    }
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
  for (const keyId of activeKeyIds) {
    expect(listKeysResponse.body.keys.map((k) => k.id)).toContain(keyId);
  }
  for (const keyId of deletedKeyIds) {
    expect(listKeysResponse.body.keys.map((k) => k.id)).not.toContain(keyId);
  }

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
}, 10_000);
