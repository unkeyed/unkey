import { RouteHarness } from "@/pkg/testutil/route-harness";
import { afterAll, afterEach, beforeAll, beforeEach, expect, test } from "vitest";
import { V1LivenessResponse } from "./v1_liveness";

let h: RouteHarness;
beforeAll(async () => {
  h = await RouteHarness.init();
});
beforeEach(async () => {
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});
afterAll(async () => {
  await h.stop();
});
test("confirms services", async () => {
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
