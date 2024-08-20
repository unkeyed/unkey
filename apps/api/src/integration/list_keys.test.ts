import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import type { V1ApisCreateApiRequest, V1ApisCreateApiResponse } from "@/routes/v1_apis_createApi";
import type { V1ApisDeleteApiRequest, V1ApisDeleteApiResponse } from "@/routes/v1_apis_deleteApi";
import type { V1ApisListKeysResponse } from "@/routes/v1_apis_listKeys";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "@/routes/v1_keys_createKey";
import type { V1KeysDeleteKeyRequest } from "@/routes/v1_keys_deleteKey";
import { expect, test } from "vitest";

test(
  "create and list keys",
  async (t) => {
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

    for (let i = 0; i < 5; i++) {
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
          enabled: true,
        },
      });
      expect(createKeyResponse.status).toEqual(200);
      expect(createKeyResponse.body.keyId).toBeDefined();
      expect(createKeyResponse.body.key).toBeDefined();
    }
    const listKeysResponse = await h.get<V1ApisListKeysResponse>({
      url: `${h.baseUrl}/v1/apis.listKeys?apiId=${createApiResponse.body.apiId}`,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${rootKey}`,
      },
    });

    expect(listKeysResponse.status).toEqual(200);
    expect(listKeysResponse.body.keys).toHaveLength(5);

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
    expect(deleteApi.status, `status mismatch - ${JSON.stringify(deleteApi)}`).toEqual(200);
  },
  { timeout: 30_000 },
);

test(
  "with revalidate",
  async (t) => {
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

    for (let i = 1; i <= 10; i++) {
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
          enabled: true,
        },
      });
      expect(createKeyResponse.status).toEqual(200);
      expect(createKeyResponse.body.keyId).toBeDefined();
      expect(createKeyResponse.body.key).toBeDefined();
      const listKeysResponse = await h.get<V1ApisListKeysResponse>({
        url: `${h.baseUrl}/v1/apis.listKeys?apiId=${createApiResponse.body.apiId}&revalidateKeysCache=true`,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${rootKey}`,
        },
      });

      expect(listKeysResponse.status).toEqual(200);
      expect(listKeysResponse.body.keys).toHaveLength(i);
    }

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
    expect(deleteApi.status, `status mismatch - ${JSON.stringify(deleteApi)}`).toEqual(200);
  },
  { timeout: 30_000 },
);

test(
  "list keys does not return revoked keys",
  async (t) => {
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
    expect(
      createApiResponse.status,
      `status mismatch - ${JSON.stringify(createApiResponse)}`,
    ).toEqual(200);
    expect(createApiResponse.body.apiId).toBeDefined();
    expect(createApiResponse.headers).toHaveProperty("unkey-request-id");

    const activeKeyIds: string[] = [];
    const deletedKeyIds: string[] = [];

    for (let i = 0; i < 10; i++) {
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
          enabled: true,
        },
      });
      expect(createKeyResponse.status).toEqual(200);
      expect(createKeyResponse.body.keyId).toBeDefined();
      expect(createKeyResponse.body.key).toBeDefined();

      if (i % 2 === 0) {
        const deleteKeyResponse = await h.post<V1KeysDeleteKeyRequest, V1KeysDeleteKeyRequest>({
          url: `${h.baseUrl}/v1/keys.deleteKey`,
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${rootKey}`,
          },
          body: {
            keyId: createKeyResponse.body.keyId,
          },
        });
        expect(
          deleteKeyResponse.status,
          `status mismatch - ${JSON.stringify(deleteKeyResponse)}`,
        ).toEqual(200);
        deletedKeyIds.push(createKeyResponse.body.keyId);
      } else {
        activeKeyIds.push(createKeyResponse.body.keyId);
      }
    }
    const listKeysResponse = await h.get<V1ApisListKeysResponse>({
      url: `${h.baseUrl}/v1/apis.listKeys?apiId=${createApiResponse.body.apiId}`,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${rootKey}`,
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
    expect(deleteApi.status, `status mismatch - ${JSON.stringify(deleteApi)}`).toEqual(200);
  },
  { timeout: 30_000 },
);
