import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { expect, test } from "vitest";
import type { V1LivenessResponse } from "./v1_liveness";

test("confirms services", async (t) => {
  const h = await IntegrationHarness.init(t);

  const res = await h.get<V1LivenessResponse>({
    url: "/v1/liveness",
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
  expect(res.body.status).toBe("we're so back");
});
