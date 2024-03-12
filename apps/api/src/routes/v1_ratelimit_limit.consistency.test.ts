import { describe, expect, test } from "vitest";

import { randomUUID } from "crypto";
import { RouteHarness } from "@/pkg/testutil/route-harness";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { V1RatelimitLimitRequest, V1RatelimitLimitResponse } from "./v1_ratelimit_limit";

describe("counts down monotonically", () => {
  test.each<{ limit: number; duration: number; n: number }>([
    { limit: 10, duration: 1_000, n: 100 },
    { limit: 10, duration: 2_000, n: 100 },
    { limit: 500, duration: 1_000, n: 100 },
    { limit: 500, duration: 60_000, n: 100 },
    // { limit: 1000, duration: 1_000, n: 250 },
  ])(
    "$limit per $duration ms @ $n runs",
    async ({ limit, duration, n }) => {
      const h = await RouteHarness.init();
      const namespace = {
        id: newId("test"),
        workspaceId: h.resources.userWorkspace.id,
        createdAt: new Date(),
        name: "namespace",
      };
      await h.db.insert(schema.ratelimitNamespaces).values(namespace);

      const identifier = randomUUID();

      const root = await h.createRootKey(["ratelimit.*.limit"]);

      let lastResponse = 10;
      for (let i = 0; i < n; i++) {
        const res = await h.post<V1RatelimitLimitRequest, V1RatelimitLimitResponse>({
          url: "/v1/ratelimit.limit",
          headers: {
            "Content-Type": "application/json",
            Authorization: `Bearer ${root.key}`,
          },
          body: {
            identifier,
            namespace: namespace.name,
            limit,
            duration,
          },
        });

        expect(res.status, `Received wrong status, res: ${JSON.stringify(res)}`).toEqual(200);
        /**
         * It should either be counting down monotonically, or be reset in a new window
         */
        expect([Math.max(0, lastResponse - 1), limit - 1]).toContain(res.body.remaining);
        lastResponse = res.body.remaining;
      }
    },
    { retry: 1, timeout: 120_000 },
  );
});
