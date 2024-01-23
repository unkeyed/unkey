import { randomUUID } from "crypto";
import { Harness } from "@/pkg/testutil/harness";
import { runSharedRoleTests } from "@/pkg/testutil/test_route_roles";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import { describe, expect, test } from "vitest";
import {
  type V1KeysUpdateKeyRequest,
  type V1KeysUpdateKeyResponse,
  registerV1KeysUpdate,
} from "./v1_keys_updateKey";

runSharedRoleTests<V1KeysUpdateKeyRequest>({
  registerHandler: registerV1KeysUpdate,
  prepareRequest: async (h) => {
    const keyId = newId("key");
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.resources.database.insert(schema.keys).values({
      id: keyId,
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
    });
    return {
      method: "POST",
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
      },
      body: {
        keyId,
        enabled: false,
      },
    };
  },
});

describe("correct roles", () => {
  test.each([
    { name: "legacy", roles: ["*"] },
    { name: "legacy and more", roles: ["*", randomUUID()] },
    { name: "wildcard api", roles: ["api.*.update_key"] },

    { name: "wildcard and more", roles: ["api.*.update_key", randomUUID()] },
    {
      name: "specific apiId",
      roles: [(apiId: string) => `api.${apiId}.update_key`],
    },
    {
      name: "specific apiId and more",
      roles: [(apiId: string) => `api.${apiId}.update_key`, randomUUID()],
    },
  ])("$name", async ({ roles }) => {
    const h = await Harness.init();
    h.useRoutes(registerV1KeysUpdate);

    const keyId = newId("key");
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await h.resources.database.insert(schema.keys).values({
      id: keyId,
      keyAuthId: h.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: h.resources.userWorkspace.id,
      createdAt: new Date(),
    });

    const root = await h.createRootKey(
      roles.map((role) => (typeof role === "string" ? role : role(h.resources.userApi.id))),
    );

    const res = await h.post<V1KeysUpdateKeyRequest, V1KeysUpdateKeyResponse>({
      url: "/v1/keys.updateKey",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: {
        keyId,
        enabled: false,
      },
    });
    expect(res.status).toEqual(200);
  });
});
