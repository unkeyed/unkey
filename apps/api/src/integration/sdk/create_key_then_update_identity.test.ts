import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { Unkey } from "@unkey/api/src/index"; // use unbundled raw esm typescript
import { newId } from "@unkey/id";
import { expect, test } from "vitest";

test("create key and then update the identity", async (t) => {
  const h = await IntegrationHarness.init(t);

  const root = await h.createRootKey([
    `api.${h.resources.userApi.id}.create_key`,
    "identity.*.create_identity",
    "identity.*.update_identity",
  ]);
  const unkey = new Unkey({ baseUrl: h.baseUrl, rootKey: root.key });

  const userId = newId("test");
  const externalId = newId("test");

  const key = await unkey.keys.create({
    apiId: h.resources.userApi.id,
    prefix: "XXX",
    externalId: externalId,
    name: "org_test",
  });


  expect(key.error).toBeUndefined();

  const identity = await unkey.identities.update({
    externalId,
    environment: "default",
    meta: {
      userId,
    },
    ratelimits: [
      {
        name: "default",
        limit: 50,
        duration: 1000,
      },
    ],
  });

  expect(identity.error).toBeUndefined();
  expect(identity.result!.meta.userId).toEqual(userId);
}, 10_000);
