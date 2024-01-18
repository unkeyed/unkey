import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";

import { runSharedRoleTests } from "@/pkg/testutil/test_route_roles";
import { LegacyKeysDeleteKeyResponse, registerLegacyKeysDelete } from "./legacy_keys_deleteKey";
runSharedRoleTests<LegacyKeysDeleteKeyResponse>({
  registerHandler: registerLegacyKeysDelete,
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
      method: "DELETE",
      url: `/v1/keys/${keyId}`,
    };
  },
});
