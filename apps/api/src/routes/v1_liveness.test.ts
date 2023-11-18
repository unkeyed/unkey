import { test, expect } from "bun:test";
import { newHonoApp } from "@/pkg/hono/app";

import { V1LivenessResponse, registerV1Liveness } from "./v1_liveness";
import { init } from "@/pkg/global";
import { testEnv } from "@/pkg/testutil/env";
import { fetchRoute } from "@/pkg/testutil/request";

test("returns 200", async () => {
  const env = testEnv();
  // @ts-ignore
  init({ env });
  const app = newHonoApp();

  registerV1Liveness(app);
  const res = await fetchRoute<never, V1LivenessResponse>(app, {
    method: "GET",
    url: "/v1/liveness",
  });

  expect(res.status).toEqual(200);

  expect(res.body.status).toEqual("we're cooking");
  expect(res.body.services.metrics).toEqual("NoopMetrics");
  expect(res.body.services.logger).toEqual("ConsoleLogger");
});
