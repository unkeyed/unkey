import { newApp } from "@/pkg/hono/app";
import { expect, test } from "vitest";

import { init } from "@/pkg/global";
import { unitTestEnv } from "@/pkg/testutil/env";
import { fetchRoute } from "@/pkg/testutil/request";
import { V1LivenessResponse, registerV1Liveness } from "./v1_liveness";

test("returns 200", async () => {
  const env = unitTestEnv.parse(process.env);
  // @ts-ignore
  init({ env });

  const app = newApp();
  registerV1Liveness(app);
  const res = await fetchRoute<never, V1LivenessResponse>(app, {
    method: "GET",
    url: "/v1/liveness",
  });

  expect(res.status).toEqual(200);
  expect(res.body.status).toEqual("we're so back");
  expect(res.body.services.metrics).toEqual("NoopMetrics");
  expect(res.body.services.logger).toEqual("ConsoleLogger");
});
