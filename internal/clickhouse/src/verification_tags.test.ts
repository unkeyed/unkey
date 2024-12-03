import { describe, expect, test } from "vitest";
import { ClickHouse } from "./index";

import { ClickHouseContainer } from "./testutil";

test.only(
  "tags are inserted correctly",
  {
    timeout: 300_000,
  },
  async (t) => {
    const container = await ClickHouseContainer.start(t, { keepContainer: true });

    const ch = new ClickHouse({ url: container.url() });

    const tagsCases: Array<Array<string>> = [
      ["key1:val1"],
      ["key1:val1", "key2:val2"],
      Array.from({ length: 100 })
        .map((_, i) => `tag_${i}`)
        .sort(),
    ];

    for (const tags of tagsCases) {
      const verification = {
        request_id: "1",
        time: Date.now(),
        workspace_id: crypto.randomUUID(),
        key_space_id: crypto.randomUUID(),
        key_id: crypto.randomUUID(),
        outcome: "VALID",
        region: "test",
        tags: tags,
      } as const;

      const { err: insertErr } = await ch.verifications.insert(verification);
      expect(insertErr).toBeUndefined();

      const latestVerifications = await ch.verifications.logs({
        workspaceId: verification.workspace_id,
        keySpaceId: verification.key_space_id,
        keyId: verification.key_id,
      });

      expect(latestVerifications.err).toBeUndefined();
      expect(latestVerifications.val!.length).toBe(1);
      expect(latestVerifications.val![0].tags).toEqual(verification.tags);
    }
  },
);

describe("materialized views", () => {
  describe("per_hour_v1", () => {
    describe("3 non-overlapping tags", () => {
      test(
        "returns 3 verifications in total",
        {
          timeout: 300_000,
        },
        async (t) => {
          const container = await ClickHouseContainer.start(t);

          const ch = new ClickHouse({ url: container.url() });

          const workspaceId = crypto.randomUUID();
          const keySpaceId = crypto.randomUUID();
          const keyId = crypto.randomUUID();
          const tags: Array<Array<string>> = [["A"], ["B"], ["C"]];

          const { err: insertErr } = await ch.verifications.insert(
            tags.map((tags, i) => ({
              request_id: i.toString(),
              time: Date.now(),
              workspace_id: workspaceId,
              key_space_id: keySpaceId,
              key_id: keyId,
              outcome: "VALID",
              region: "test",
              tags: tags,
            })),
          );
          expect(insertErr).toBeUndefined();
          const allVerifications = await ch.verifications.logs({
            workspaceId,
            keySpaceId,
            keyId,
          });

          /**
           * Assert all of the rows have been written to clickhouse
           */
          expect(allVerifications.err).toBeUndefined();
          expect(allVerifications.val!.length).toBe(tags.length);

          /**
           * Wait for materialized views to be updated
           */
          await new Promise((r) => setTimeout(r, 5000));

          const hourly = await ch.verifications.perHour({
            workspaceId,
            keySpaceId,
            keyId,
            start: Date.now() - 60 * 60 * 1000,
            end: Date.now(),
          });
          console.log(JSON.stringify({ hourly }, null, 2));

          expect(hourly.err).toBeUndefined();

          const total = hourly.val!.reduce((sum, hour) => {
            return sum + hour.count;
          }, 0);
          expect(total).toBe(4);
        },
      );
    });
  });
});
