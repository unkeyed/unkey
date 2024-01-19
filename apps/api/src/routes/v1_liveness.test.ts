import { Harness } from "@/pkg/testutil/harness";
import { expect, test } from "vitest";
import { V1LivenessResponse, registerV1Liveness } from "./v1_liveness";

test("returns 200", async () => {
  const h = await Harness.init();
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
