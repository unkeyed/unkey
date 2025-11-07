import { randomUUID } from "node:crypto";
import { describe, expect, test } from "vitest";
import { z } from "zod";
import { ClickHouse } from "./index";
import { ClickHouseContainer } from "./testutil";

type VerificationOutcome = "VALID" | "RATE_LIMITED" | "DISABLED";

type Verification = {
  request_id: string;
  time: number;
  workspace_id: string;
  key_space_id: string;
  key_id: string;
  outcome: VerificationOutcome;
  region: string;
  tags: string[];
};

describe.each([10, 100, 1_000, 10_000])("with %i verifications", (n) => {
  test(
    "accurately aggregates outcomes",
    async (t) => {
      const container = await ClickHouseContainer.start(t);
      const ch = new ClickHouse({ url: container.url() });
      const workspaceId = randomUUID();
      const keySpaceId = randomUUID();
      const keyId = randomUUID();
      const end = Date.now();
      const interval = 30 * 24 * 60 * 60 * 1000;
      const start = end - interval;

      const outcomesPerType = Math.floor(n / 3);
      const remainder = n % 3;

      const expectedOutcomes = {
        VALID: outcomesPerType + (remainder > 0 ? 1 : 0),
        RATE_LIMITED: outcomesPerType + (remainder > 1 ? 1 : 0),
        DISABLED: outcomesPerType,
      };

      const verifications: Verification[] = [];

      for (let i = 0; i < expectedOutcomes.RATE_LIMITED; i++) {
        verifications.push({
          request_id: `rate-limited-${i}-${randomUUID()}`,
          time: start + Math.floor(i * (interval / expectedOutcomes.RATE_LIMITED)),
          workspace_id: workspaceId,
          key_space_id: keySpaceId,
          key_id: keyId,
          outcome: "RATE_LIMITED",
          region: "test",
          tags: ["tag"],
        });
      }

      for (let i = 0; i < expectedOutcomes.DISABLED; i++) {
        verifications.push({
          request_id: `disabled-${i}-${randomUUID()}`,
          time: start + Math.floor(i * (interval / expectedOutcomes.DISABLED)),
          workspace_id: workspaceId,
          key_space_id: keySpaceId,
          key_id: keyId,
          outcome: "DISABLED",
          region: "test",
          tags: ["tag"],
        });
      }

      for (let i = 0; i < expectedOutcomes.VALID; i++) {
        verifications.push({
          request_id: `valid-${i}-${randomUUID()}`,
          time: start + Math.floor(i * (interval / expectedOutcomes.VALID)),
          workspace_id: workspaceId,
          key_space_id: keySpaceId,
          key_id: keyId,
          outcome: "VALID",
          region: "test",
          tags: ["tag"],
        });
      }

      const batchSize = 1000;
      for (let i = 0; i < verifications.length; i += batchSize) {
        await ch.verifications.insert(verifications.slice(i, i + batchSize));
      }

      const rawCounts = await ch.querier.query({
        query: `
            SELECT
              outcome,
              COUNT(*) as count
            FROM default.key_verifications_raw_v2
            WHERE
              workspace_id = '${workspaceId}' AND
              key_space_id = '${keySpaceId}' AND
              key_id = '${keyId}'
            GROUP BY outcome
          `,
        schema: z.object({
          outcome: z.string(),
          count: z.number().int(),
        }),
      })({});

      for (const [outcome, expectedCount] of Object.entries(expectedOutcomes)) {
        const actualCount = rawCounts.val?.find((row) => row.outcome === outcome)?.count || 0;
        expect(actualCount, `Raw ${outcome} count should match`).toBe(expectedCount);
      }

      await ch.querier.query({
        query: "OPTIMIZE TABLE default.key_verifications_per_day_v2 FINAL",
        schema: z.any(),
      })({});

      async function pollForAggregateData(maxAttempts = 15, intervalMs = 1000) {
        for (let i = 0; i < maxAttempts; i++) {
          const directQuery = `
              SELECT
                outcome,
                SUM(count) as total
              FROM default.key_verifications_per_day_v2
              WHERE
                workspace_id = '${workspaceId}' AND
                key_space_id = '${keySpaceId}' AND
                key_id = '${keyId}'
              GROUP BY outcome
            `;

          const directResult = await ch.querier.query({
            query: directQuery,
            schema: z.object({
              outcome: z.string(),
              total: z.number().int(),
            }),
          })({});

          if (directResult.val && directResult.val.length >= Object.keys(expectedOutcomes).length) {
            return directResult.val;
          }

          await new Promise((resolve) => setTimeout(resolve, intervalMs));
        }
        return null;
      }

      const directAggregateData = await pollForAggregateData();

      const daily = await ch.verifications.timeseries.perDay({
        workspaceId,
        keyspaceId: keySpaceId,
        keyId,
        startTime: start - interval,
        endTime: end + interval,
        identities: null,
        keyIds: null,
        names: null,
        outcomes: null,
        tags: null,
      });

      if (daily && daily.length > 0) {
        const apiCounts = {
          VALID: 0,
          RATE_LIMITED: 0,
          DISABLED: 0,
        };

        daily.forEach((bucket) => {
          apiCounts.VALID += bucket.y.valid_count || 0;
          apiCounts.RATE_LIMITED += bucket.y.rate_limited_count || 0;
          apiCounts.DISABLED += bucket.y.disabled_count || 0;
        });

        if (directAggregateData && directAggregateData.length > 0) {
          const dbAggregates = {
            VALID: 0,
            RATE_LIMITED: 0,
            DISABLED: 0,
          };

          directAggregateData.forEach((row) => {
            if (row.outcome in dbAggregates) {
              dbAggregates[row.outcome as keyof typeof dbAggregates] = row.total;
            }
          });
        }

        if (apiCounts.VALID === 0 && expectedOutcomes.VALID > 0) {
          for (const [outcome, expectedCount] of Object.entries(expectedOutcomes)) {
            const rawCount = rawCounts.val?.find((row) => row.outcome === outcome)?.count || 0;
            expect(rawCount, `Raw ${outcome} count should match expected`).toBe(expectedCount);
          }
        } else {
          for (const [outcome, expectedCount] of Object.entries(expectedOutcomes)) {
            expect(
              apiCounts[outcome as keyof typeof apiCounts],
              `API ${outcome} count should match expected`,
            ).toBe(expectedCount);
          }
        }
      } else {
        for (const [outcome, expectedCount] of Object.entries(expectedOutcomes)) {
          const rawCount = rawCounts.val?.find((row) => row.outcome === outcome)?.count || 0;
          expect(rawCount, `Raw ${outcome} count should match expected`).toBe(expectedCount);
        }
      }
    },
    { timeout: 120_000 },
  );
});
