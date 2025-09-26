import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { eq, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "./v1_keys_createKey";

runCommonRouteTests<V1KeysCreateKeyRequest>({
  prepareRequest: async (rh) => {
    const apiId = newId("api");
    await rh.db.primary.insert(schema.apis).values({
      id: apiId,
      name: randomUUID(),
      workspaceId: rh.resources.userWorkspace.id,
    });
    return {
      method: "POST",
      url: "/v1/keys.createKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        apiId,
        byteLength: 16,
      },
    };
  },
});

describe("correct roles", () => {
  describe.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
    { name: "wildcard api", roles: ["api.*.create_key"] },

    { name: "wildcard and more", roles: ["api.*.create_key", randomUUID()] },
    {
      name: "specific apiId",
      roles: [(apiId: string) => `api.${apiId}.create_key`],
    },
    {
      name: "specific apiId and more",
      roles: [(apiId: string) => `api.${apiId}.create_key`, randomUUID()],
    },
  ])("$name", ({ roles }) => {
    test("returns 200", async (t) => {
      const h = await IntegrationHarness.init(t);
      const keyAuthId = newId("test");
      await h.db.primary.insert(schema.keyAuth).values({
        id: keyAuthId,
        workspaceId: h.resources.userWorkspace.id,
      });

      const apiId = newId("api");
      await h.db.primary.insert(schema.apis).values({
        id: apiId,
        name: randomUUID(),
        workspaceId: h.resources.userWorkspace.id,
        authType: "key",
        keyAuthId,
      });

      const root = await h.createRootKey(
        roles.map((role) => (typeof role === "string" ? role : role(apiId))),
      );

      const res = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: "/v1/keys.createKey",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId,
        },
      });
      expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    });
  });
});

test("cannot encrypt without permissions", async (t) => {
  const h = await IntegrationHarness.init(t);

  await h.db.primary
    .update(schema.keyAuth)
    .set({
      storeEncryptedKeys: true,
    })
    .where(eq(schema.keyAuth.id, h.resources.userKeyAuth.id));

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  const res = await h.post<V1KeysCreateKeyRequest, { error: { code: string } }>({
    url: "/v1/keys.createKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      recoverable: true,
    },
  });

  t.expect(res.status, `expected 403, received: ${JSON.stringify(res, null, 2)}`).toBe(403);
  t.expect(res.body.error.code).toEqual("INSUFFICIENT_PERMISSIONS");
});

test("cannot create role without permissions", async (t) => {
  const h = await IntegrationHarness.init(t);

  await h.db.primary
    .update(schema.keyAuth)
    .set({
      storeEncryptedKeys: true,
    })
    .where(eq(schema.keyAuth.id, h.resources.userKeyAuth.id));

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  const res = await h.post<V1KeysCreateKeyRequest, { error: { code: string } }>({
    url: "/v1/keys.createKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      roles: ["r1"],
    },
  });

  t.expect(res.status, `expected 403, received: ${JSON.stringify(res, null, 2)}`).toBe(403);
  t.expect(res.body.error.code).toEqual("INSUFFICIENT_PERMISSIONS");
});

test("cannot create permission without permissions", async (t) => {
  const h = await IntegrationHarness.init(t);

  await h.db.primary
    .update(schema.keyAuth)
    .set({
      storeEncryptedKeys: true,
    })
    .where(eq(schema.keyAuth.id, h.resources.userKeyAuth.id));

  const root = await h.createRootKey([`api.${h.resources.userApi.id}.create_key`]);

  const res = await h.post<V1KeysCreateKeyRequest, { error: { code: string } }>({
    url: "/v1/keys.createKey",
    headers: {
      "Content-Type": "application/json",
      Authorization: `Bearer ${root.key}`,
    },
    body: {
      apiId: h.resources.userApi.id,
      permissions: ["p1"],
    },
  });

  t.expect(res.status, `expected 403, received: ${JSON.stringify(res, null, 2)}`).toBe(403);
  t.expect(res.body.error.code).toEqual("INSUFFICIENT_PERMISSIONS");
});
