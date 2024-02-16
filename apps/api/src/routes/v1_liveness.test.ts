import { Harness } from "@/pkg/testutil/route-harness";
import { expect, test } from "vitest";
import { V1LivenessResponse, registerV1Liveness } from "./v1_liveness";

test("returns 200", async () => {
  using h = new Harness();
  await h.seed();
  h.useRoutes(registerV1Liveness);
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
