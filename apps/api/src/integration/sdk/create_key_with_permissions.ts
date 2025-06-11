import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { Unkey, and } from "@unkey/api/src/index"; // use unbundled raw esm typescript
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { expect, test } from "vitest";

test("create with permissions", async (t) => {
  const h = await IntegrationHarness.init(t);

  const permissions = ["domain.create", "dns.record.create", "domain.delete"];
  await h.db.primary.insert(schema.permissions).values(
    permissions.map((name) => ({
      id: newId("test"),
      name,
      slug: name,
      workspaceId: h.resources.userWorkspace.id,
    })),
  );

  const { key: rootKey } = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  const sdk = new Unkey({
    baseUrl: h.baseUrl,
    rootKey: rootKey,
  });

  const create = await sdk.keys.create({
    apiId: h.resources.userApi.id,
    permissions: ["domain.create", "dns.record.create"],
  });
  expect(create.error).toBeUndefined();
  const key = create.result!.key;

  const verify = await sdk.keys.verify({
    apiId: h.resources.userApi.id,
    key,
    authorization: {
      permissions: and("domain.create", "dns.record.create"),
    },
  });

  expect(verify.error).toBeUndefined();
  expect(verify.result).toBeDefined();

  expect(verify.result!.valid).toBe(true);
}, 10_000);
