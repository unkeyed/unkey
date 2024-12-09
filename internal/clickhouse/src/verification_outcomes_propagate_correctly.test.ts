import { describe, expect, test } from "vitest";
import { ClickHouse } from "./index";

import { randomUUID } from "node:crypto";
import { z } from "zod";
import { ClickHouseContainer } from "./testutil";

describe.each([10, 100, 1_000, 10_000, 100_000])("with %i verifications", (n) => {
  test(
    "accurately aggregates outcomes",
    async (t) => {
      const container = await ClickHouseContainer.start(t);

      const ch = new ClickHouse({ url: container.url() });

      const workspaceId = randomUUID();
      const keySpaceId = randomUUID();
      const keyId = randomUUID();

      const end = Date.now();
      const interval = 90 * 24 * 60 * 60 * 1000; // 90 days
      const start = end - interval;
      const outcomes = {
        VALID: 0,
        RATE_LIMITED: 0,
        DISABLED: 0,
      };
      const verifications = Array.from({ length: n }).map((_) => {
        const outcome = Object.keys(outcomes)[
          Math.floor(Math.random() * Object.keys(outcomes).length)
        ] as keyof typeof outcomes;
        outcomes[outcome]++;
        return {
          request_id: randomUUID(),
          time: Math.round(Math.random() * (end - start + 1) + start),
          workspace_id: workspaceId,
          key_space_id: keySpaceId,
          key_id: keyId,
          outcome,
          region: "test",
          tags: ["tag"],
        };
      });

      for (let i = 0; i < verifications.length; i += 1000) {
        await ch.verifications.insert(verifications.slice(i, i + 1000));
      }

      // give clickhouse time to write to all tables
      // await new Promise(r => setTimeout(r, 60_000))

      const count = await ch.querier.query({
        query: "SELECT count(*) as count FROM verifications.raw_key_verifications_v1",
        schema: z.object({ count: z.number().int() }),
      })({});
      expect(count.err).toBeUndefined();
      expect(count.val!.at(0)!.count).toBe(n);

      const hourly = await ch.verifications.perHour({
        workspaceId,
        keySpaceId,
        keyId,
        start: start - interval,
        end,
      });
      expect(hourly.err).toBeUndefined();

      const daily = await ch.verifications.perDay({
        workspaceId,
        keySpaceId,
        keyId,
        start: start - interval,
        end,
      });
      expect(daily.err).toBeUndefined();

      const monthly = await ch.verifications.perMonth({
        workspaceId,
        keySpaceId,
        keyId,
        start: start - interval,
        end,
      });
      expect(monthly.err).toBeUndefined();

      for (const buckets of [hourly.val!, daily.val!, monthly.val!]) {
        let total = 0;
        const sumByOutcome = buckets.reduce(
          (acc, bucket) => {
            total += bucket.count;
            if (!acc[bucket.outcome]) {
              acc[bucket.outcome] = 0;
            }
            acc[bucket.outcome] += bucket.count;
            return acc;
          },
          {} as Record<keyof typeof outcomes, number>,
        );

        expect(total).toBe(n);

        for (const [k, v] of Object.entries(outcomes)) {
          expect(sumByOutcome[k]).toEqual(v);
        }
      }
    },
    { timeout: 120_000 },
  );
});
