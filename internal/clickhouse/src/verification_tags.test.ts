import { describe, expect, test } from "vitest";
import { ClickHouse } from "./index";

import { ClickHouseContainer } from "./testutil";

const waitForMaterializedViews = (ms = 5000) => new Promise((resolve) => setTimeout(resolve, ms));

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

      const latestVerifications = await ch.verifications.latest({
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
  for (const mv of ["hour", "day", "month"]) {
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
          const keyspaceId = crypto.randomUUID();
          const keyId = crypto.randomUUID();

          // Insert test verifications
          const { err: insertErr } = await ch.verifications.insert(
            verificationsWithTags.map((tags, i) => ({
              request_id: i.toString(),
              time: Date.now(),
              workspace_id: workspaceId,
              key_space_id: keyspaceId,
              key_id: keyId,
              outcome: "VALID",
              region: "test",
              tags: tags,
            })),
          );
          expect(insertErr).toBeUndefined();

          // Verify records were inserted using latest
          const allVerifications = await ch.verifications.latest({
            workspaceId,
            keySpaceId: keyspaceId,
            keyId,
            limit: 50,
          });

          expect(allVerifications.err).toBeUndefined();
          expect(allVerifications.val!.length).toBe(verificationsWithTags.length);

          // Wait for materialized views to update
          await waitForMaterializedViews();

          // Get the appropriate materialized view function based on the timeseries granularity
          const timeseriesFunction = {
            hour: ch.verifications.timeseries.perHour,
            day: ch.verifications.timeseries.perDay,
            month: ch.verifications.timeseries.perMonth,
          }[mv];

          // Make sure the function exists
          expect(timeseriesFunction).toBeDefined();

          // Query the materialized view
          const mvRes = await timeseriesFunction?.({
            workspaceId,
            keyspaceId,
            keyId,
            startTime: Date.now() - 60 * 24 * 60 * 60 * 1000, // 60 days ago
            endTime: Date.now() + 10000, // Add buffer to current time
            names: null, // Required parameter
            identities: null, // Required parameter
            keyIds: null, // Required parameter
            outcomes: null, // Required parameter
            tags: null, // Required parameter
          });

          // Calculate total verification count from all data points
          const totalVerifications = mvRes!.reduce((sum, dataPoint) => {
            // The total property in y contains the count
            return sum + dataPoint.y.total;
          }, 0);

          // Verify the total count matches our verification count
          expect(totalVerifications).toBe(verificationsWithTags.length);
        },
      );
    });
  }
});
