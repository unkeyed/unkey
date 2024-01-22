import { randomUUID } from "crypto";
import { Harness } from "@/pkg/testutil/harness";
import { runSharedRoleTests } from "@/pkg/testutil/test_route_roles";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { describe, expect, test } from "vitest";
import {
  type V1KeysCreateKeyRequest,
  V1KeysCreateKeyResponse,
  registerV1KeysCreateKey,
} from "./v1_keys_createKey";

runSharedRoleTests<V1KeysCreateKeyRequest>({
  registerHandler: registerV1KeysCreateKey,
  prepareRequest: async (h) => {
    const apiId = newId("api");
    await h.db.insert(schema.apis).values({
      id: apiId,
      name: randomUUID(),
      workspaceId: h.resources.userWorkspace.id,
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
  test.each([
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
  ])("$name", async ({ roles }) => {
    const h = await Harness.init();
    h.useRoutes(registerV1KeysCreateKey);

    const keyAuthId = newId("keyAuth");
    await h.db.insert(schema.keyAuth).values({
      id: keyAuthId,
      workspaceId: h.resources.userWorkspace.id,
    });

    const apiId = newId("api");
    await h.db.insert(schema.apis).values({
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
    expect(res.status).toEqual(200);
  });
});
