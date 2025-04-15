import { describe, expect, test } from "vitest";
import { ClickHouse } from "./index";

import { ClickHouseContainer } from "./testutil";

test(
  "tags are inserted correctly",
  {
    timeout: 300_000,
  },
  async (t) => {
    const container = await ClickHouseContainer.start(t, {
      keepContainer: true,
    });

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
        limit: 1,
      });

      expect(latestVerifications.err).toBeUndefined();
      expect(latestVerifications.val!.length).toBe(1);
      expect(latestVerifications.val![0].tags).toEqual(verification.tags);
    }
  },
);

describe("materialized views", () => {
  for (const mv of ["per_hour", "per_day", "per_month"]) {
    describe(mv, () => {
      const verificationsWithTags: Array<Array<string>> = [
        [],
        ["A"],
        ["B"],
        ["C"],
        ["A", "B"],
        ["A", "B", "D"],
        ["B", "A", "C"],
      ];

      test(
        `returns ${verificationsWithTags.length} verifications in total`,
        {
          timeout: 300_000,
        },
        async (t) => {
          const container = await ClickHouseContainer.start(t);

          const ch = new ClickHouse({ url: container.url() });

          const workspaceId = crypto.randomUUID();
          const keySpaceId = crypto.randomUUID();
          const keyId = crypto.randomUUID();

          const { err: insertErr } = await ch.verifications.insert(
            verificationsWithTags.map((tags, i) => ({
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
            limit: 50,
          });

          /**
           * Assert all of the rows have been written to clickhouse
           */
          expect(allVerifications.err).toBeUndefined();
          expect(allVerifications.val!.length).toBe(verificationsWithTags.length);

          /**
           * Wait for materialized views to be updated
           */
          await new Promise((r) => setTimeout(r, 5000));

          const q = {
            per_hour: ch.verifications.perHour,
            per_day: ch.verifications.perDay,
            per_month: ch.verifications.perMonth,
          }[mv]!;
          const mvRes = await q({
            workspaceId,
            keySpaceId,
            keyId,
            start: Date.now() - 60 * 24 * 60 * 60 * 1000,
            end: Date.now(),
          });

          expect(mvRes.err).toBeUndefined();

          const total = mvRes.val!.reduce((sum, v) => {
            return sum + v.count;
          }, 0);
          expect(total).toBe(verificationsWithTags.length);
        },
      );
    });
  }
});
