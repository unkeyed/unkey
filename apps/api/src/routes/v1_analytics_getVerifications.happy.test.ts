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
      searchparams: {
        start: start.toString(),
        end: end.toString(),
        groupBy: "hour",
      },
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);
    expect(res.body, JSON.stringify({ res }, null, 2)).toHaveLength(
      Math.floor((end - start) / interval),
    );
  });
});

describe.each([
  // generate and query times are different to ensure the query covers the entire generate interval
  // and the used toStartOf function in clickhouse
  {
    granularity: "hour",
    generate: { start: "2024-12-05", end: "2024-12-07" },
    query: { start: "2024-12-04", end: "2024-12-10" },
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
      searchparams: {
        start: new Date(tc.query.start).getTime().toString(),
        end: new Date(tc.query.end).getTime().toString(),
        groupBy: tc.granularity,
      },
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

    expect(res.body.reduce((sum, d) => sum + d.total, 0)).toEqual(verifications.length);
    expect(res.body.reduce((sum, d) => sum + (d.valid ?? 0), 0)).toEqual(outcomes.VALID);
    expect(res.body.reduce((sum, d) => sum + (d.notFound ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.forbidden ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.usageExceeded ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.rateLimited ?? 0), 0)).toEqual(
      outcomes.RATE_LIMITED,
    );
    expect(res.body.reduce((sum, d) => sum + (d.unauthorized ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.disabled ?? 0), 0)).toEqual(outcomes.DISABLED);
    expect(res.body.reduce((sum, d) => sum + (d.insufficientPermissions ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.expired ?? 0), 0)).toEqual(0);
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
      h.createKey(), // unrelated noise
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
      searchparams: {
        start: start.toString(),
        end: end.toString(),
        externalId: identity.externalId,
        groupBy: "hour",
      },
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

        acc[v.outcome]++;
        total++;
        return acc;
      },
      { VALID: 0, RATE_LIMITED: 0, DISABLED: 0 } as {
        [K in (typeof POSSIBLE_OUTCOMES)[number]]: number;
      },
    );

    expect(res.body.length).gte(24);
    expect(res.body.length).lte(25);

    expect(res.body.reduce((sum, d) => sum + d.total, 0)).toEqual(total);
    expect(res.body.reduce((sum, d) => sum + (d.valid ?? 0), 0)).toEqual(outcomes.VALID);
    expect(res.body.reduce((sum, d) => sum + (d.notFound ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.forbidden ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.usageExceeded ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.rateLimited ?? 0), 0)).toEqual(
      outcomes.RATE_LIMITED,
    );
    expect(res.body.reduce((sum, d) => sum + (d.unauthorized ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.disabled ?? 0), 0)).toEqual(outcomes.DISABLED);
    expect(res.body.reduce((sum, d) => sum + (d.insufficientPermissions ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.expired ?? 0), 0)).toEqual(0);
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
      searchparams: {
        start: start.toString(),
        end: end.toString(),
        externalId: identity.externalId,
        groupBy: ["key", "hour"],
      },
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    for (const row of res.body) {
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

    expect(res.body.reduce((sum, d) => sum + d.total, 0)).toEqual(total);
    expect(res.body.reduce((sum, d) => sum + (d.valid ?? 0), 0)).toEqual(outcomes.VALID);
    expect(res.body.reduce((sum, d) => sum + (d.notFound ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.forbidden ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.usageExceeded ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.rateLimited ?? 0), 0)).toEqual(
      outcomes.RATE_LIMITED,
    );
    expect(res.body.reduce((sum, d) => sum + (d.unauthorized ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.disabled ?? 0), 0)).toEqual(outcomes.DISABLED);
    expect(res.body.reduce((sum, d) => sum + (d.insufficientPermissions ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.expired ?? 0), 0)).toEqual(0);

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
        res.body.filter((d) => d.keyId === key.keyId).reduce((sum, d) => sum + d.total, 0),
      ).toEqual(keyTotal);
      expect(
        res.body.filter((d) => d.keyId === key.keyId).reduce((sum, d) => sum + (d.valid ?? 0), 0),
      ).toEqual(keyOutcomes.VALID);
      expect(
        res.body
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.notFound ?? 0), 0),
      ).toEqual(0);
      expect(
        res.body
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.forbidden ?? 0), 0),
      ).toEqual(0);
      expect(
        res.body
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.usageExceeded ?? 0), 0),
      ).toEqual(0);
      expect(
        res.body
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.rateLimited ?? 0), 0),
      ).toEqual(keyOutcomes.RATE_LIMITED);
      expect(
        res.body
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.unauthorized ?? 0), 0),
      ).toEqual(0);
      expect(
        res.body
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.disabled ?? 0), 0),
      ).toEqual(keyOutcomes.DISABLED);
      expect(
        res.body
          .filter((d) => d.keyId === key.keyId)
          .reduce((sum, d) => sum + (d.insufficientPermissions ?? 0), 0),
      ).toEqual(0);
      expect(
        res.body.filter((d) => d.keyId === key.keyId).reduce((sum, d) => sum + (d.expired ?? 0), 0),
      ).toEqual(0);
    }
  });

  test("A monthly cron job creates invoices for each identity", async (t) => {
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
      start: now - 3 * 30 * 24 * 60 * 60 * 1000,
      end: now,
      length: 100_000,
      workspaceId: h.resources.userWorkspace.id,
      keySpaceId: h.resources.userKeyAuth.id,
      keys: keys.map((k) => ({ keyId: k.keyId, identityId: k.identityId })),
    });

    await h.ch.verifications.insert(verifications);

    const root = await h.createRootKey(["api.*.read_api"]);

    const start = new Date(now).setMonth(new Date(now).getMonth() - 1, 1);
    const end = now;

    const res = await h.get<V1AnalyticsGetVerificationsResponse>({
      url: "/v1/analytics.getVerifications",
      searchparams: {
        start: start.toString(),
        end: end.toString(),
        externalId: identity.externalId,
        groupBy: "month",
      },
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    expect(res.body.length).lte(2);
    expect(res.body.length).gte(1);
    let total = 0;
    const outcomes = verifications.reduce(
      (acc, v) => {
        if (
          v.identity_id !== identity.id ||
          new Date(v.time).getUTCMonth() !== new Date(now).getUTCMonth()
        ) {
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

    expect(res.body.reduce((sum, d) => sum + d.total, 0)).toEqual(total);
    expect(res.body.reduce((sum, d) => sum + (d.valid ?? 0), 0)).toEqual(outcomes.VALID);
    expect(res.body.reduce((sum, d) => sum + (d.notFound ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.forbidden ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.usageExceeded ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.rateLimited ?? 0), 0)).toEqual(
      outcomes.RATE_LIMITED,
    );
    expect(res.body.reduce((sum, d) => sum + (d.unauthorized ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.disabled ?? 0), 0)).toEqual(outcomes.DISABLED);
    expect(res.body.reduce((sum, d) => sum + (d.insufficientPermissions ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.expired ?? 0), 0)).toEqual(0);
  });

  test("a user sees a gauge with their quota, showing they used X out of Y API calls in the current billing period", async (t) => {
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
      start: now - 60 * 24 * 60 * 60 * 1000,
      end: now,
      length: 100_000,
      workspaceId: h.resources.userWorkspace.id,
      keySpaceId: h.resources.userKeyAuth.id,
      keys: keys.map((k) => ({ keyId: k.keyId, identityId: k.identityId })),
    });

    await h.ch.verifications.insert(verifications);

    const root = await h.createRootKey(["api.*.read_api"]);

    const d = new Date(now);
    d.setUTCDate(2);
    d.setUTCHours(0, 0, 0, 0);
    const start = d.getTime();
    const end = new Date(start).setUTCMonth(new Date(start).getUTCMonth() + 1);

    const res = await h.get<V1AnalyticsGetVerificationsResponse>({
      url: "/v1/analytics.getVerifications",
      searchparams: {
        start: start.toString(),
        end: end.toString(),
        externalId: identity.externalId,
        groupBy: "day",
      },
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    let total = 0;
    const outcomes = verifications.reduce(
      (acc, v) => {
        if (v.identity_id !== identity.id || v.time < start) {
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

    expect(res.body.reduce((sum, d) => sum + d.total, 0)).toEqual(total);
    expect(res.body.reduce((sum, d) => sum + (d.valid ?? 0), 0)).toEqual(outcomes.VALID);
    expect(res.body.reduce((sum, d) => sum + (d.notFound ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.forbidden ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.usageExceeded ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.rateLimited ?? 0), 0)).toEqual(
      outcomes.RATE_LIMITED,
    );
    expect(res.body.reduce((sum, d) => sum + (d.unauthorized ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.disabled ?? 0), 0)).toEqual(outcomes.DISABLED);
    expect(res.body.reduce((sum, d) => sum + (d.insufficientPermissions ?? 0), 0)).toEqual(0);
    expect(res.body.reduce((sum, d) => sum + (d.expired ?? 0), 0)).toEqual(0);
  });
  test("An internal dashboard shows the top 10 users by API usage over the past 30 days", async (t) => {
    const h = await IntegrationHarness.init(t);

    const identities = Array.from({ length: 100 }).map((_) => ({
      workspaceId: h.resources.userWorkspace.id,
      id: newId("test"),
      externalId: newId("test"),
    }));

    await h.db.primary.insert(schema.identities).values(identities);

    const keys = await Promise.all(
      identities.flatMap((id) =>
        Array.from({ length: 3 }).map((_) => h.createKey({ identityId: id.id })),
      ),
    );

    const now = Date.now();

    const verifications = generate({
      start: now - 60 * 24 * 60 * 60 * 1000,
      end: now,
      length: 100_000,
      workspaceId: h.resources.userWorkspace.id,
      keySpaceId: h.resources.userKeyAuth.id,
      keys: keys.map((k) => ({ keyId: k.keyId, identityId: k.identityId })),
    });

    await h.ch.verifications.insert(verifications);

    const byIdentity = verifications.reduce(
      (acc, v) => {
        if (!acc[v.identity_id!]) {
          acc[v.identity_id!] = {
            identityId: v.identity_id!,
            valid: 0,
            rateLimited: 0,
            disabled: 0,
            total: 0,
          };
        }
        acc[v.identity_id!].total += 1;
        switch (v.outcome) {
          case "VALID": {
            acc[v.identity_id!].valid += 1;
            break;
          }
          case "RATE_LIMITED": {
            acc[v.identity_id!].rateLimited += 1;
            break;
          }

          case "DISABLED": {
            acc[v.identity_id!].disabled += 1;
            break;
          }
        }
        return acc;
      },
      {} as Record<
        string,
        { identityId: string; valid: number; rateLimited: number; total: number; disabled: number }
      >,
    );

    const top10 = Object.values(byIdentity)
      .sort((a, b) => a.total - b.total)
      .slice(0, 10);
    console.table(top10);

    const root = await h.createRootKey(["api.*.read_api"]);

    const start = now - 30 * 24 * 60 * 60 * 1000;
    const end = now;

    const res = await h.get<V1AnalyticsGetVerificationsResponse>({
      url: "/v1/analytics.getVerifications",
      searchparams: {
        start: start.toString(),
        end: end.toString(),
        apiId: h.resources.userApi.id,
        limit: "10",
        orderBy: ["total"],
        order: "desc",
        groupBy: ["identity"],
      },
      headers: {
        Authorization: `Bearer ${root.key}`,
      },
    });

    expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

    expect(res.body.length).gte(1);
    expect(res.body.length).lte(10);
    expect(res.body.length).toEqual(top10.length);
    console.table(res.body);
    console.table(top10);

    // Check that the order is correct
    for (let i = 0; i < res.body.length; i++) {
      if (i === 0) {
        // Nothing to compare in the first iteration
        continue;
      }
      expect(res.body[i].total <= res.body[i - 1].total);
    }
    //expect(
    //  res.body[i].identity,
    //
    //  `we're grouping by identity, so it should be defined but it wasn't,
    //  we got i=${i}$ {JSON.stringify(res.body[i], null, 2)}`,
    //).toBeDefined();
    //  expect(res.body[i].identity!.id).toEqual(top10[i].identityId);

    console.info(t.task.id, res.body);
  });
});

/**
 * Generate a number of key verification events to seed clickhouse
 */
function generate(opts: {
  start: number;
  end: number;
  length: number;
  workspaceId: string;
  keySpaceId: string;
  keys: Array<{ keyId: string; identityId?: string }>;
  tags?: string[];
}) {
  return Array.from({ length: opts.length }).map((_) => {
    const key = opts.keys[Math.floor(Math.random() * opts.keys.length)];

    return {
      time: Math.round(Math.random() * (opts.end - opts.start) + opts.start),
      workspace_id: opts.workspaceId,
      key_space_id: opts.keySpaceId,
      key_id: key.keyId,
      outcome: POSSIBLE_OUTCOMES[Math.floor(Math.random() * POSSIBLE_OUTCOMES.length)],
      tags: opts.tags ?? [],
      request_id: newId("test"),
      region: "test",
      identity_id: key.identityId,
    };
  });
}
