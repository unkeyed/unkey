import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";

import { runCommonRouteTests } from "@/pkg/testutil/common-tests";
import type { V1KeysDeleteKeyRequest } from "./v1_keys_deleteKey";

runCommonRouteTests<V1KeysDeleteKeyRequest>({
  prepareRequest: async (rh) => {
    const keyId = newId("test");
    const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
    await rh.db.primary.insert(schema.keys).values({
      id: keyId,
      keyAuthId: rh.resources.userKeyAuth.id,
      hash: await sha256(key),
      start: key.slice(0, 8),
      workspaceId: rh.resources.userWorkspace.id,
      createdAtM: Date.now(),
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
