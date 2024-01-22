import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";

import { runSharedRoleTests } from "@/pkg/testutil/test_route_roles";
import { V1KeysDeleteKeyRequest, registerV1KeysDeleteKey } from "./v1_keys_deleteKey";

runSharedRoleTests<V1KeysDeleteKeyRequest>({
  registerHandler: registerV1KeysDeleteKey,
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
      url: "/v1/keys.deleteKey",
      body: {
        keyId,
      },
      headers: {
        "Content-Type": "application/json",
      },
    };
  },
});
