import { expect, test } from "vitest";
import { ClickHouse } from "./index";

import { randomUUID } from "node:crypto";
import { ClickHouseContainer } from "./testutil";

test(
  "returns the correct amount of billable verifications",
  {
    timeout: 300_000,
  },
  async (t) => {
    const container = await ClickHouseContainer.start(t);

    const ch = new ClickHouse({ url: container.url() });

    const workspaceId = randomUUID();
    const keySpaceId = randomUUID();
    const keyId = randomUUID();
    const now = new Date();
    const year = now.getUTCFullYear();
    const month = now.getUTCMonth() + 1; // 1 = January
    const endTime = now.getTime();
    const startTime = now.setUTCDate(1);

    const outcomes = ["VALID", "RATE_LIMITED", "DISABLED"] as const;
    let valid = 0;
    for (let i = 0; i < 100; i++) {
      const verifications = new Array(10_000).fill(null).map(() => {
        const outcome = outcomes[Math.floor(Math.random() * outcomes.length)];
        if (outcome === "VALID") {
          valid++;
        }
        return {
          workspace_id: workspaceId,
          key_space_id: keySpaceId,
          key_id: keyId,
          outcome,
          time: Math.floor(startTime + Math.random() * (endTime - startTime)),
          region: "test",
          request_id: randomUUID(),
          tags: Array.from({ length: Math.floor(Math.random() * 10) }).map((i) => `tag_${i}`),
        };
      });

      const { err } = await ch.verifications.insert(verifications);
      expect(err).toBeUndefined();
    }
    // give clickhouse time to process all writes and update the materialized views
    await new Promise((r) => setTimeout(r, 10_000));

    const billableVerifications = await ch.billing.billableVerifications({
      workspaceId,
      year,
      month,
    });

    expect(billableVerifications).toBe(valid);
  },
);
