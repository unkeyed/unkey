import { Analytics } from "@/pkg/analytics";
import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import type { V1ApisCreateApiRequest, V1ApisCreateApiResponse } from "@/routes/v1_apis_createApi";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "@/routes/v1_keys_createKey";
import { V1KeysDeleteKeyRequest, V1KeysDeleteKeyResponse } from "@/routes/v1_keys_deleteKey";
import { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "@/routes/v1_keys_verifyKey";
import { expect, test } from "vitest";

test("create, verify keys, check tb for verifications, delete key", async (t) => {
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
      prefix: "test",
      apiId: createApiResponse.body.apiId,
      byteLength: 16,
      enabled: true,
      ownerId: "test-integration",
    },
  });
  expect(key.status).toEqual(200);
  expect(key.body.key).toBeDefined();
  expect(key.body.keyId).toBeDefined();

  const key2 = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: `${h.baseUrl}/v1/keys.createKey`,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      prefix: "test",
      apiId: createApiResponse.body.apiId,
      byteLength: 16,
      enabled: true,
      ownerId: "test-integration",
    },
  });
  expect(key2.status).toEqual(200);
  expect(key2.body.key).toBeDefined();
  expect(key2.body.keyId).toBeDefined();
  const valid2 = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: `${h.baseUrl}/v1/keys.verifyKey`,
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      key: key2.body.key,
      apiId: createApiResponse.body.apiId,
    },
  });
  expect(valid2.status).toEqual(200);
  expect(valid2.body.valid).toEqual(true);

  const key3 = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
    url: `${h.baseUrl}/v1/keys.createKey`,
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${rootKey}`,
    },
    body: {
      prefix: "test",
      apiId: createApiResponse.body.apiId,
      byteLength: 16,
      enabled: true,
      ownerId: "test-integration",
    },
  });
  expect(key3.status).toEqual(200);
  expect(key3.body.key).toBeDefined();
  expect(key3.body.keyId).toBeDefined();
  const valid3 = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
    url: `${h.baseUrl}/v1/keys.verifyKey`,
    headers: {
      "Content-Type": "application/json",
    },
    body: {
      key: key2.body.key,
      apiId: createApiResponse.body.apiId,
    },
  });
  expect(valid3.status).toEqual(200);
  expect(valid3.body.valid).toEqual(true);
  //check tb to see if it has verifications
  const analytics = new Analytics();
  const verifications = await analytics.getVerificationsByOwnerIdDaily({
    workspaceId: h.resources.userWorkspace.id,
    ownerId: "test-integration",
    apiId: createApiResponse.body.apiId,
  });

  expect(verifications).toEqual({
    data: [],
    meta: [],
  });
  expect(verifications).toBeDefined();

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
