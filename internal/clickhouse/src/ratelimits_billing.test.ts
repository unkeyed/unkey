import { expect, test } from "vitest";
import { ClickHouse } from "./index";

import { randomUUID } from "node:crypto";
import { ClickHouseContainer } from "./testutil";

test(
  "returns the correct amount of billable ratelimits",
  {
    timeout: 300_000,
  },
  async (t) => {
    const container = await ClickHouseContainer.start(t);

    const ch = new ClickHouse({ url: container.url() });

    const workspaceId = randomUUID();
    const namespaceId = randomUUID();
    const now = new Date();
    const year = now.getUTCFullYear();
    const month = now.getUTCMonth() + 1; // 1 = January
    const endTime = now.getTime();
    const startTime = now.setUTCDate(1);

    let billable = 0;
    for (let i = 0; i < 100; i++) {
      const ratelimits = new Array(10_000).fill(null).map(() => {
        const passed = Math.random() > 0.2;
        if (passed) {
          billable++;
        }
        return {
          workspace_id: workspaceId,
          namespace_id: namespaceId,
          identifier: randomUUID(),
          passed,
          time: Math.floor(startTime + Math.random() * (endTime - startTime)),
          request_id: randomUUID(),
        };
      });

      await ch.ratelimits.insert(ratelimits);
    }
    // give clickhouse time to process all writes and update the materialized views
    await new Promise((r) => setTimeout(r, 10_000));

    const billableRatelimits = await ch.billing.billableRatelimits({
      workspaceId,
      year,
      month,
    });

    expect(billableRatelimits).toBe(billable);
  },
);
