import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { Unkey } from "@unkey/api/src/index"; // use unbundled raw esm typescript
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { expect, test } from "vitest";

test("1 per 10 seconds", async (t) => {
  const h = await IntegrationHarness.init(t);

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  await h.db.primary.insert(schema.roles).values({
    id: newId("test"),
    workspaceId: h.resources.userWorkspace.id,
    name: "role",
  });

  const sdk = new Unkey({
    baseUrl: h.baseUrl,
    rootKey: root.key,
  });

  const { result: key, error: createKeyError } = await sdk.keys.create({
    apiId: h.resources.userApi.id,
    ownerId: "ownerId",
    roles: ["role"],
    ratelimit: {
      limit: 1,
      duration: 10000,
      async: false,
    },
  });
  expect(createKeyError).toBeUndefined();
  expect(key).toBeDefined();

  const { result: firstVerify, error: firstVerifyError } = await sdk.keys.verify({
    apiId: h.resources.userApi.id,
    key: key!.key,
    ratelimit: {
      cost: 1,
    },
  });
  expect(firstVerifyError).toBeUndefined();
  expect(firstVerify).toBeDefined();
  expect(firstVerify!.code).toBe("VALID");

  const { result: secondVerify, error: secondVerifyError } = await sdk.keys.verify({
    apiId: h.resources.userApi.id,
    key: key!.key,
  });
  expect(secondVerifyError).toBeUndefined();
  expect(secondVerify).toBeDefined();
  expect(secondVerify!.code).toBe("RATE_LIMITED");
});
