import { RouteHarness } from "@/pkg/testutil/route-harness";
import { afterEach, beforeEach, expect, test } from "vitest";
import { V1LivenessResponse, registerV1Liveness } from "./v1_liveness";

let h: RouteHarness;
beforeEach(async () => {
  h = new RouteHarness();
  h.useRoutes(registerV1Liveness);
  await h.seed();
});
afterEach(async () => {
  await h.teardown();
});
test("returns 200", async () => {
  const res = await h.get<V1LivenessResponse>({
    url: "/v1/liveness",
  });

  expect(res).toMatchObject({
    status: 200,
    body: {
      status: "we're so back",
      services: {
        metrics: "NoopMetrics",
        logger: "ConsoleLogger",
      },
    },
  });
});
