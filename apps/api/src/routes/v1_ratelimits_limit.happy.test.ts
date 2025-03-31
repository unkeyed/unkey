import { expect, test } from "vitest";

import { randomUUID } from "node:crypto";
import { IntegrationHarness } from "@/pkg/testutil/integration-harness";
import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import type { V1RatelimitLimitRequest, V1RatelimitLimitResponse } from "./v1_ratelimits_limit";
test("setting cost=0 returns the limit without modifying", async (t) => {
  const h = await IntegrationHarness.init(t);
  const namespace = {
    id: newId("test"),
    workspaceId: h.resources.userWorkspace.id,
    createdAtM: Date.now(),
    name: "namespace",
  };
  await h.db.primary.insert(schema.ratelimitNamespaces).values(namespace);

  const identifier = randomUUID();

  const root = await h.createRootKey(["ratelimit.*.limit"]);

  const request = {
    identifier,
    namespace: namespace.name,
    limit: 10,
    duration: 10000,
    async: false,
  };

  for (let i = 0; i < 5; i++) {
    const res = await h.post<V1RatelimitLimitRequest, V1RatelimitLimitResponse>({
      url: "/v1/ratelimits.limit",
      headers: {
        "Content-Type": "application/json",
        Authorization: `Bearer ${root.key}`,
      },
      body: { ...request, cost: 0 },
    });

    expect(res.status, `Received wrong status, res: ${JSON.stringify(res)}`).toEqual(200);
    expect(res.body.remaining).toEqual(request.limit);
  }
});
