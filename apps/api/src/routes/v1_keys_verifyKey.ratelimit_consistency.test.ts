import { describe, expect, test } from "vitest";

import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { schema } from "@unkey/db";
import { sha256 } from "@unkey/hash";
import { newId } from "@unkey/id";
import { KeyV1 } from "@unkey/keys";
import type { V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse } from "./v1_keys_verifyKey";

describe.each<{ limit: number; duration: number; n: number }>([
  { limit: 10, duration: 1_000, n: 100 },
  { limit: 10, duration: 2_000, n: 100 },
  { limit: 500, duration: 1_000, n: 100 },
  { limit: 500, duration: 60_000, n: 100 },
  // { limit: 1000, duration: 1_000, n: 250 },
])("$limit per $duration ms @ $n runs", async ({ limit, duration, n }) => {
  test(
    "counts down monotonically",
    async (t) => {
      const h = await IntegrationHarness.init(t);

      const key = new KeyV1({ prefix: "test", byteLength: 16 }).toString();
      await h.db.primary.insert(schema.keys).values({
        id: newId("test"),
        keyAuthId: h.resources.userKeyAuth.id,
        hash: await sha256(key),
        start: key.slice(0, 8),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        ratelimitLimit: limit,
        ratelimitDuration: duration,
        ratelimitAsync: false,
      });

      let lastResponse = limit;
      for (let i = 0; i < n; i++) {
        const res = await h.post<V1KeysVerifyKeyRequest, V1KeysVerifyKeyResponse>({
          url: "/v1/keys.verifyKey",
          headers: {
            "Content-Type": "application/json",
          },
          body: {
            key,
            apiId: h.resources.userApi.id,
          },
        });

        expect(res.status, `Received wrong status, res: ${JSON.stringify(res)}`).toEqual(200);
        expect(res.body.ratelimit).toBeDefined();
        /**
         * It should either be counting down monotonically, or be reset in a new window
         */
        expect([Math.max(0, lastResponse - 1), limit - 1]).toContain(res.body.ratelimit!.remaining);
        lastResponse = res.body.ratelimit!.remaining;
      }
    },
    { timeout: 120_000 },
  );
});
