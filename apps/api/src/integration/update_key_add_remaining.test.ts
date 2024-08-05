import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import type { V1ApisCreateApiRequest, V1ApisCreateApiResponse } from "@/routes/v1_apis_createApi";
import type { V1ApisDeleteApiRequest, V1ApisDeleteApiResponse } from "@/routes/v1_apis_deleteApi";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "@/routes/v1_keys_createKey";
import type {
  V1KeysUpdateRemainingRequest,
  V1KeysUpdateRemainingResponse,
} from "@/routes/v1_keys_updateRemaining";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "@/routes/v1_keys_verifyKey";
import { describe, expect, test } from "vitest";

describe("some", () => {
  test("update a key's remaining limit", async (t) => {
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
        remaining: 5,
        enabled: true,
      },
    });
    expect(createKeyResponse.status).toEqual(200);

    for (let i = 4; i >= 0; i--) {
      const valid = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
        url: `${h.baseUrl}/v1/keys.verifyKey`,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${rootKey}`,
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
        Authorization: `Bearer ${rootKey}`,
      },
      body: {
        apiId: createApiResponse.body.apiId,
        key: createKeyResponse.body.key,
      },
    });
    expect(invalid.status).toEqual(200);
    expect(invalid.body.valid).toBe(false);
    expect(invalid.body.remaining).toEqual(0);

    const updateKeyResponse = await h.post<
      V1KeysUpdateRemainingRequest,
      V1KeysUpdateRemainingResponse
    >({
      url: `${h.baseUrl}/v1/keys.updateRemaining`,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${rootKey}`,
      },
      body: {
        keyId: createKeyResponse.body.keyId,
        op: "increment",
        value: 5,
      },
    });

    expect(updateKeyResponse.status).toEqual(200);
    expect(updateKeyResponse.body.remaining).toEqual(5);

    const validAfterUpdate = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: `${h.baseUrl}/v1/keys.verifyKey`,
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${rootKey}`,
      },
      body: {
        apiId: createApiResponse.body.apiId,
        key: createKeyResponse.body.key,
      },
    });
    expect(validAfterUpdate.status).toEqual(200);
    expect(validAfterUpdate.body.valid).toBe(true);
    expect(validAfterUpdate.body.remaining).toEqual(4);

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
  });
});
