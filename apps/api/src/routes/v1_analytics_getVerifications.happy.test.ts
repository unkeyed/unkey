import { IntegrationHarness } from "src/pkg/testutil/integration-harness";

import { schema } from "@unkey/db";
import { newId } from "@unkey/id";
import { describe, expect, test } from "vitest";
import { z } from "zod";
import type { V1AnalyticsGetVerificationsResponse } from "./v1_analytics_getVerifications";

const POSSIBLE_OUTCOMES = ["VALID", "RATE_LIMITED", "DISABLED"] as const;

describe("with no data", () => {
  test("returns an array with one element per interval", async (t) => {
    const h = await IntegrationHarness.init(t);

    const end = Date.now();
    const interval = 60 * 60 * 1000; // hour
    const start = end - 30 * interval;

    const root = await h.createRootKey(["api.*.read_api"]);
    const res = await h.get<V1AnalyticsGetVerificationsResponse>({
      url: "/v1/analytics.getVerifications",
      searchparams: new URLSearchParams({
        start: start.toString(),
        end: end.toString(),
        granularity: "hour",
      }),
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    expect(res.body.data).toHaveLength(Math.floor((end - start) / interval));
  });
});

describe.each([
  // generate and query times are different to ensure the query covers the entire generate interval
  // and the used toStartOf function in clickhouse
  {
    granularity: "hour",
    generate: { start: "2024-12-05", end: "2024-12-07" },
    query: { start: "2024-11-29", end: "2024-12-10" },
  },
  {
    granularity: "day",
    generate: { start: "2024-12-05", end: "2024-12-07" },
    query: { start: "2024-12-01", end: "2024-12-10" },
  },
  {
    granularity: "month",
    generate: { start: "2024-10-1", end: "2025-10-12" },
    query: { start: "2023-12-01", end: "2026-12-10" },
  },
])("per $granularity", (tc) => {
  test("all verifications are accounted for", async (t) => {
    const h = await IntegrationHarness.init(t);

    const verifications = generate({
      start: new Date(tc.generate.start).getTime(),
      end: new Date(tc.generate.end).getTime(),
      length: 100_000,
      workspaceId: h.resources.userWorkspace.id,
      keySpaceId: h.resources.userKeyAuth.id,
      keys: Array.from({ length: 3 }).map(() => ({ keyId: newId("test") })),
    });

    await h.ch.verifications.insert(verifications);

    const inserted = await h.ch.querier.query({
      query:
        "SELECT COUNT(*) AS count from verifications.raw_key_verifications_v1 WHERE workspace_id={workspaceId:String}",
      params: z.object({ workspaceId: z.string() }),
      schema: z.object({ count: z.number() }),
    })({
      workspaceId: h.resources.userWorkspace.id,
    });
    expect(inserted.val!.at(0)?.count).toEqual(verifications.length);

    const root = await h.createRootKey(["api.*.read_api"]);

    const res = await h.get<V1AnalyticsGetVerificationsResponse>({
      url: "/v1/analytics.getVerifications",
      searchparams: new URLSearchParams({
        start: new Date(tc.query.start).getTime().toString(),
        end: new Date(tc.query.end).getTime().toString(),
        granularity: tc.granularity,
      }),
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    const outcomes = verifications.reduce(
      (acc, v) => {
        if (!acc[v.outcome]) {
          acc[v.outcome] = 0;
        }
        acc[v.outcome]++;
        return acc;
      },
      {} as { [K in (typeof POSSIBLE_OUTCOMES)[number]]: number },
    );

    console.table(outcomes);

    expect(res.body.data.reduce((sum, d) => sum + d.outcomes.total, 0)).toEqual(
      verifications.length,
    );
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.valid ?? 0), 0)).toEqual(
      outcomes.VALID,
    );
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.notFound ?? 0), 0)).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.forbidden ?? 0), 0)).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.usageExceeded ?? 0), 0)).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.rateLimited ?? 0), 0)).toEqual(
      outcomes.RATE_LIMITED,
    );
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.unauthorited ?? 0), 0)).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.disabled ?? 0), 0)).toEqual(
      outcomes.DISABLED,
    );
    expect(
      res.body.data.reduce((sum, d) => sum + (d.outcomes.insufficientPermissions ?? 0), 0),
    ).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.expired ?? 0), 0)).toEqual(0);
  });
});

describe("RFC scenarios", () => {
  test("a user's usage over the past 24h for 2 keys", async (t) => {
    const h = await IntegrationHarness.init(t);

    const identity = {
      workspaceId: h.resources.userWorkspace.id,
      id: newId("test"),
      externalId: newId("test"),
    };

    await h.db.primary.insert(schema.identities).values(identity);

    const keys = await Promise.all([
      h.createKey({ identityId: identity.id }),
      h.createKey({ identityId: identity.id }),
      h.createKey({ identityId: identity.id }),
      h.createKey(),
    ]);

    const now = Date.now();

    const verifications = generate({
      start: now - 12 * 60 * 60 * 1000,
      end: now,
      length: 100_000,
      workspaceId: h.resources.userWorkspace.id,
      keySpaceId: h.resources.userKeyAuth.id,
      keys: keys.map((k) => ({ keyId: k.keyId, identityId: k.identityId })),
    });

    await h.ch.verifications.insert(verifications);

    const root = await h.createRootKey(["api.*.read_api"]);

    const start = now - 24 * 60 * 60 * 1000;
    const end = now;

    const res = await h.get<V1AnalyticsGetVerificationsResponse>({
      url: "/v1/analytics.getVerifications",
      searchparams: new URLSearchParams({
        start: start.toString(),
        end: end.toString(),
        granularity: "hour",
        externalId: identity.externalId,
      }),
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    let total = 0;
    const outcomes = verifications.reduce(
      (acc, v) => {
        if (v.identity_id !== identity.id) {
          return acc;
        }

        if (!acc[v.outcome]) {
          acc[v.outcome] = 0;
        }
        acc[v.outcome]++;
        total++;
        return acc;
      },
      {} as { [K in (typeof POSSIBLE_OUTCOMES)[number]]: number },
    );

    console.info(
      res.body.data.map((b) =>
        JSON.stringify({
          ...b,
          time: new Date(b.time),
        }),
      ),
    );
    expect(res.body.data.length).gte(24);
    expect(res.body.data.length).lte(25);

    expect(res.body.data.reduce((sum, d) => sum + d.outcomes.total, 0)).toEqual(total);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.valid ?? 0), 0)).toEqual(
      outcomes.VALID,
    );
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.notFound ?? 0), 0)).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.forbidden ?? 0), 0)).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.usageExceeded ?? 0), 0)).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.rateLimited ?? 0), 0)).toEqual(
      outcomes.RATE_LIMITED,
    );
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.unauthorited ?? 0), 0)).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.disabled ?? 0), 0)).toEqual(
      outcomes.DISABLED,
    );
    expect(
      res.body.data.reduce((sum, d) => sum + (d.outcomes.insufficientPermissions ?? 0), 0),
    ).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.expired ?? 0), 0)).toEqual(0);
  });

  test("daily usage breakdown for a user per key in the current month", async (t) => {
    const h = await IntegrationHarness.init(t);

    const identity = {
      workspaceId: h.resources.userWorkspace.id,
      id: newId("test"),
      externalId: newId("test"),
    };

    await h.db.primary.insert(schema.identities).values(identity);

    const keys = await Promise.all([
      h.createKey({ identityId: identity.id }),
      h.createKey({ identityId: identity.id }),
      h.createKey({ identityId: identity.id }),
      h.createKey(),
    ]);

    const now = Date.now();

    const verifications = generate({
      start: now - 12 * 60 * 60 * 1000,
      end: now,
      length: 100_000,
      workspaceId: h.resources.userWorkspace.id,
      keySpaceId: h.resources.userKeyAuth.id,
      keys: keys.map((k) => ({ keyId: k.keyId, identityId: k.identityId })),
    });

    await h.ch.verifications.insert(verifications);

    const root = await h.createRootKey(["api.*.read_api"]);

    const start = now - 24 * 60 * 60 * 1000;
    const end = now;

    const res = await h.get<V1AnalyticsGetVerificationsResponse>({
      url: "/v1/analytics.getVerifications",
      searchparams: new URLSearchParams({
        start: start.toString(),
        end: end.toString(),
        granularity: "hour",
        externalId: identity.externalId,
        groupBy: "key",
      }),
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    for (const row of res.body.data) {
      expect(row.keyId).toBeDefined();
    }

    let total = 0;
    const outcomes = verifications.reduce(
      (acc, v) => {
        if (v.identity_id !== identity.id) {
          return acc;
        }

        acc[v.outcome]++;
        total++;
        return acc;
      },
      { VALID: 0, DISABLED: 0, RATE_LIMITED: 0 } as {
        [K in (typeof POSSIBLE_OUTCOMES)[number]]: number;
      },
    );

    expect(res.body.data.reduce((sum, d) => sum + d.outcomes.total, 0)).toEqual(total);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.valid ?? 0), 0)).toEqual(
      outcomes.VALID,
    );
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.notFound ?? 0), 0)).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.forbidden ?? 0), 0)).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.usageExceeded ?? 0), 0)).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.rateLimited ?? 0), 0)).toEqual(
      outcomes.RATE_LIMITED,
    );
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.unauthorited ?? 0), 0)).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.disabled ?? 0), 0)).toEqual(
      outcomes.DISABLED,
    );
    expect(
      res.body.data.reduce((sum, d) => sum + (d.outcomes.insufficientPermissions ?? 0), 0),
    ).toEqual(0);
    expect(res.body.data.reduce((sum, d) => sum + (d.outcomes.expired ?? 0), 0)).toEqual(0);

    //   Per Key
    for (const key of keys.filter((k) => k.identityId)) {
      let keyTotal = 0;
      const keyOutcomes = verifications.reduce(
        (acc, v) => {
          if (v.key_id !== key.keyId) {
            return acc;
          }

          acc[v.outcome]++;
          keyTotal++;
          return acc;
        },
        { VALID: 0, DISABLED: 0, RATE_LIMITED: 0 } as {
          [K in (typeof POSSIBLE_OUTCOMES)[number]]: number;
        },
      );

      expect(
        res.body.data
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + d.outcomes.total, 0),
      ).toEqual(keyTotal);
      expect(
        res.body.data
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.outcomes.valid ?? 0), 0),
      ).toEqual(keyOutcomes.VALID);
      expect(
        res.body.data
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.outcomes.notFound ?? 0), 0),
      ).toEqual(0);
      expect(
        res.body.data
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.outcomes.forbidden ?? 0), 0),
      ).toEqual(0);
      expect(
        res.body.data
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.outcomes.usageExceeded ?? 0), 0),
      ).toEqual(0);
      expect(
        res.body.data
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.outcomes.rateLimited ?? 0), 0),
      ).toEqual(keyOutcomes.RATE_LIMITED);
      expect(
        res.body.data
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.outcomes.unauthorited ?? 0), 0),
      ).toEqual(0);
      expect(
        res.body.data
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.outcomes.disabled ?? 0), 0),
      ).toEqual(keyOutcomes.DISABLED);
      expect(
        res.body.data
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.outcomes.insufficientPermissions ?? 0), 0),
      ).toEqual(0);
      expect(
        res.body.data
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.outcomes.expired ?? 0), 0),
      ).toEqual(0);
    }
  });
});

function generate(opts: {
  start: number;
  end: number;
  length: number;
  workspaceId: string;
  keySpaceId: string;
  keys: Array<{ keyId: string; identityId?: string }>;
  tags?: string[];
}) {
  const key = opts.keys[Math.floor(Math.random() * opts.keys.length)];
  return Array.from({ length: opts.length }).map((_) => ({
    time: Math.round(Math.random() * (opts.end - opts.start) + opts.start),
    workspace_id: opts.workspaceId,
    key_space_id: opts.keySpaceId,
    key_id: key.keyId,
    outcome: POSSIBLE_OUTCOMES[Math.floor(Math.random() * POSSIBLE_OUTCOMES.length)],
    tags: opts.tags ?? [],
    request_id: newId("test"),
    region: "test",
    identity_id: key.identityId,
  }));
}
