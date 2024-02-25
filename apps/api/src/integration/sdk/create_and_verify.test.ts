import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { type Flatten, Unkey, or } from "@unkey/api/src/index"; // use unbundled raw esm typescript
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { afterEach, beforeEach, expect, test } from "vitest";

let h: IntegrationHarness;

beforeEach(async () => {
  h = new IntegrationHarness();
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});
test("create with roles and permissions", async () => {
  await h.seed();

  type Resources = {
    domain: "create" | "delete" | "read";
    dns: {
      record: "create" | "read" | "delete";
    };
  };
  type Permissions = Flatten<Resources, ".">;

  const roleId = newId("role");
  await h.db.insert(schema.roles).values({
    id: roleId,
    name: "domain.manager",
    workspaceId: h.resources.userWorkspace.id,
  });

  for (const name of ["domain.create", "dns.record.create", "domain.delete"]) {
    const permissionId = newId("permission");
    await h.db.insert(schema.permissions).values({
      id: permissionId,
      name,
      workspaceId: h.resources.userWorkspace.id,
    });

    await h.db.insert(schema.rolesPermissions).values({
      roleId,
      permissionId,
      workspaceId: h.resources.userWorkspace.id,
    });
  }

  const { key: rootKey } = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  const sdk = new Unkey({
    baseUrl: h.baseUrl,
    rootKey: rootKey,
  });

  const create = await sdk.keys.create({
    apiId: h.resources.userApi.id,
    roles: ["domain.manager"],
  });
  expect(create.error).toBeUndefined();
  const key = create.result!.key;

  const verify = await sdk.keys.verify<Permissions>({
    apiId: h.resources.userApi.id,
    key,
    authorization: {
      permissions: or("domain.create", "dns.record.create"),
    },
  });

  expect(verify.error).toBeUndefined();
  expect(verify.result).toBeDefined();

  expect(verify.result!.valid).toBe(true);
});
