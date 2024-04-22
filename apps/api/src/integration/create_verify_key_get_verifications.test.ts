import { Analytics } from "@/pkg/analytics";
import { ConsoleLogger } from "@/pkg/logging";
import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import type { V1ApisCreateApiRequest, V1ApisCreateApiResponse } from "@/routes/v1_apis_createApi";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "@/routes/v1_keys_createKey";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "@/routes/v1_keys_verifyKey";
import { expect, test } from "vitest";
test("create, verify keys, check tb for verifications, delete key", async (t) => {
  const h = await IntegrationHarness.init(t);
  const { key: rootKey } = await h.createRootKey(["*"]);
  const logger = new ConsoleLogger({
    defaultFields: { environment: process.env.ENVIRONMENT },
  });
  const clickhouse =
    process.env.CLICKHOUSE_URL && process.env.CLICKHOUSE_USERNAME && process.env.CLICKHOUSE_PASSWORD
      ? {
          url: process.env.CLICKHOUSE_URL,
          username: process.env.CLICKHOUSE_USERNAME,
          password: process.env.CLICKHOUSE_PASSWORD,
        }
      : undefined;
  if (clickhouse) {
    logger.info("Using clickhouse");
  }
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

  const keyCount = 3;
  const keys = [];
  for (let index = 0; index < keyCount; index++) {
    keys[index] = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
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
    expect(keys[index].status).toEqual(200);
    expect(keys[index].body.key).toBeDefined();
    expect(keys[index].body.keyId).toBeDefined();
    const valid = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
      url: `${h.baseUrl}/v1/keys.verifyKey`,
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        key: keys[index].body.key,
        apiId: createApiResponse.body.apiId,
      },
    });
    expect(valid.status).toEqual(200);
    expect(valid.body.valid).toEqual(true);
  }
  //check tb to see if it has verifications
  const analytics = new Analytics({ tinybirdToken: process.env.TINYBIRD_TOKEN, clickhouse });
  const verifications = await analytics.getVerificationsByOwnerIdDaily({
    workspaceId: h.resources.userWorkspace.id,
    ownerId: "test-integration",
    apiId: createApiResponse.body.apiId,
  });

  expect(verifications).toEqual({
    data: [],
    meta: [],
  });
});
