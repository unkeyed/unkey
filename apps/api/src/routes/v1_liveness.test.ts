import { RouteHarness } from "@/pkg/testutil/route-harness";
import { expect, test } from "vitest";
import type { V1LivenessResponse } from "./v1_liveness";

test("confirms services", async (t) => {
  const h = await RouteHarness.init(t);

  const res = await h.get<V1LivenessResponse>({
    url: "/v1/liveness",
  });

  expect(res).toMatchObject({
    status: 200,
    body: {
      status: "we're so back",
      services: {
        // metrics: "NoopMetrics",
        logger: "ConsoleLogger",
        ratelimit: "DurableRateLimiter",
        usagelimit: "DurableUsageLimiter",
        analytics: "NoopTinybird",
      },
    },
  });
});
