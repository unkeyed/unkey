import { RouteHarness } from "@/pkg/testutil/route-harness";
import { expect, test } from "vitest";
import { V1LivenessResponse } from "./v1_liveness";

test("confirms services", async () => {
  const h = await RouteHarness.init();

  const res = await h.get<V1LivenessResponse>({
    url: "/v1/liveness",
  });

  console.log(res);
  expect(res).toMatchObject({
    status: 200,
    body: {
      status: "we're so back",
      services: {
        metrics: "NoopMetrics",
        logger: "ConsoleLogger",
        ratelimit: "DurableRateLimiter",
        usagelimit: "DurableUsageLimiter",
        analytics: "NoopTinybird",
      },
    },
  });
});
