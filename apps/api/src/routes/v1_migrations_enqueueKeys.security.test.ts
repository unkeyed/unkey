import { randomUUID } from "node:crypto";
import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import { eq, schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { describe, expect, test } from "vitest";
import type {
  V1MigrationsEnqueueKeysRequest,
  V1MigrationsEnqueueKeysResponse,
} from "./v1_migrations_enqueueKeys";

runCommonRouteTests<V1MigrationsEnqueueKeysRequest>({
  prepareRequest: async (rh) => {
    const apiId = newId("api");
    await rh.db.primary.insert(schema.apis).values({
      id: apiId,
      name: randomUUID(),
      workspaceId: rh.resources.userWorkspace.id,
    });
    return {
      method: "POST",
      url: "/v1/migrations.enqueueKeys",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        apiId,
        migrationId: "helo",
        keys: [
          {
            start: "start_",
            hash: {
              value: "hash",
              variant: "sha256_base64",
            },
          },
        ],
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
    test("returns 202", async (t) => {
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

      const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
        url: "/v1/migrations.enqueueKeys",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId,
          migrationId: "helo",
          keys: [
            {
              start: "start_",
              hash: {
                value: crypto.randomUUID(),
                variant: "sha256_base64",
              },
            },
          ],
        },
      });
      expect(res.status, `expected 202, received: ${JSON.stringify(res, null, 2)}`).toBe(202);
    });
  });
});

describe("encrypting requires permissions", () => {
  describe.each([
    {
      name: "without permissions",
      status: 403,
      roles: [],
    },
    {
      name: "with specific permission",
      status: 202,
      roles: [
        (apiId: string) => `api.${apiId}.encrypt_key`,
        (apiId: string) => `api.${apiId}.create_key`,
      ],
    },
    {
      name: "with wildcard permission",
      status: 202,
      roles: ["api.*.encrypt_key", "api.*.create_key"],
    },
    {
      name: "with mixed permission",
      status: 202,
      roles: [(apiId: string) => `api.${apiId}.encrypt_key`, "api.*.create_key"],
    },
  ])("$name", ({ status, roles }) => {
    test("encrypting requires permissions", async (t) => {
      const h = await IntegrationHarness.init(t);
      await h.db.primary
        .update(schema.keyAuth)
        .set({
          storeEncryptedKeys: true,
        })
        .where(eq(schema.keyAuth.id, h.resources.userKeyAuth.id));

      const root = await h.createRootKey(
        roles.map((role) => (typeof role === "string" ? role : role(h.resources.userApi.id))),
      );

      const res = await h.post<V1MigrationsEnqueueKeysRequest, V1MigrationsEnqueueKeysResponse>({
        url: "/v1/migrations.enqueueKeys",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${root.key}`,
        },
        body: {
          apiId: h.resources.userApi.id,

          migrationId: "helo",
          keys: [
            {
              start: "start_",
              plaintext: crypto.randomUUID(),
            },
          ],
        },
      });

      expect(res.status, `expected ${status}, received: ${JSON.stringify(res, null, 2)}`).toBe(
        status,
      );
    });
  });
});
