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
        apiId: h.resources.userApi.id,
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
  test("all verifications are accounted for", { timeout: 120_000 }, async (t) => {
    const h = await IntegrationHarness.init(t);

    const verifications = generate({
      start: new Date(tc.generate.start).getTime(),
      end: new Date(tc.generate.end).getTime(),
      length: 100_000,
      workspaceId: h.resources.userWorkspace.id,
      keySpaceId: h.resources.userKeyAuth.id,
      keys: Array.from({ length: 3 }).map(() => ({ keyId: newId("test") })),
    });

    const insert = await h.ch.verifications.insert(verifications);
    expect(insert.err).toEqual(undefined);

    const inserted = await h.ch.querier.query({
      query:
        "SELECT COUNT(*) AS count from verifications.raw_key_verifications_v1 WHERE workspace_id={workspaceId:String}",
      params: z.object({ workspaceId: z.string() }),
      schema: z.object({ count: z.number() }),
    })({
      workspaceId: h.resources.userWorkspace.id,
    });
    expect(inserted.err).toEqual(undefined);
    expect(inserted.val!.at(0)?.count).toEqual(verifications.length);

    const root = await h.createRootKey(["api.*.read_api"]);

    const res = await h.get<V1AnalyticsGetVerificationsResponse>({
      url: "/v1/analytics.getVerifications",
      searchparams: {
        start: new Date(tc.query.start).getTime().toString(),
        end: new Date(tc.query.end).getTime().toString(),
        groupBy: tc.granularity,
        apiId: h.resources.userApi.id,
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
  test("a user's usage over the past 24h for 2 keys", { timeout: 120_000 }, async (t) => {
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
        apiId: h.resources.userApi.id,
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

  test(
    "daily usage breakdown for a user per key in the current month",
    { timeout: 120_000 },
    async (t) => {
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
          apiId: h.resources.userApi.id,
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
          res.body
            .filter((d) => d.keyId === key.keyId)
            .reduce((sum, d) => sum + (d.expired ?? 0), 0),
        ).toEqual(0);
      }
    },
  );

  test("A monthly cron job creates invoices for each identity", { timeout: 120_000 }, async (t) => {
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
        apiId: h.resources.userApi.id,
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

  test(
    "a user sees a gauge with their quota, showing they used X out of Y API calls in the current billing period",
    { timeout: 120_000 },
    async (t) => {
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
          apiId: h.resources.userApi.id,
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
    },
  );

  test(
    "An internal dashboard shows the top 10 users by API usage over the past 30 days",
    { timeout: 120_000 },
    async (t) => {
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

      const start = now - 30 * 24 * 60 * 60 * 1000;
      const end = now;

      await h.ch.verifications.insert(verifications);

      const byIdentity = verifications.reduce(
        (acc, v) => {
          if (toStartOfHour(v.time) < start || toStartOfHour(v.time) > end) {
            return acc;
          }
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
          {
            identityId: string;
            valid: number;
            rateLimited: number;
            total: number;
            disabled: number;
          }
        >,
      );

      const top15 = Object.values(byIdentity)
        .sort((a, b) => b.total - a.total)
        .slice(0, 15);

      const root = await h.createRootKey(["api.*.read_api"]);

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
      expect(res.body.length).lte(top15.length);

      for (let i = 0; i < res.body.length; i++) {
        if (i === 0) {
          // Nothing to compare in the first iteration
          continue;
        }
        // Order should be desc
        expect(res.body[i].total <= res.body[i - 1].total);

        expect(
          res.body[i].identity,
          `we're grouping by identity, so it should be defined but it wasn't,
        we got i=${i}$ ${JSON.stringify(res.body[i], null, 2)}`,
        ).toBeDefined();
        expect(res.body[i].total).toEqual(top15[i].total);
      }

      for (const row of res.body) {
        const actual = top15.find((i) => i.identityId === row.identity!.id);
        expect(actual).toBeDefined();
        expect(row.total).toEqual(actual!.total);
      }
    },
  );
});

test("filter by tag", { timeout: 120_000 }, async (t) => {
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

  const tags = [["a", "b"], ["a"], [], ["b", "c"]];

  const verifications = tags.flatMap((tags) =>
    generate({
      start: now - 60 * 24 * 60 * 60 * 1000,
      end: now,
      length: 100_000,
      workspaceId: h.resources.userWorkspace.id,
      keySpaceId: h.resources.userKeyAuth.id,
      keys: keys.map((k) => ({ keyId: k.keyId, identityId: k.identityId })),
      tags,
    }),
  );

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
      apiId: h.resources.userApi.id,
      tag: "a",
    },
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const verificationsForTag = verifications.reduce(
    (acc, v) => {
      if (toStartOfHour(v.time) < start || toStartOfHour(v.time) > end || !v.tags.includes("a")) {
        return acc;
      }
      acc.total += 1;
      switch (v.outcome) {
        case "VALID": {
          acc.valid += 1;
          break;
        }
        case "DISABLED": {
          acc.disabled += 1;
          break;
        }
        case "RATE_LIMITED": {
          acc.rateLimited += 1;
          break;
        }
      }

      return acc;
    },
    { total: 0, valid: 0, disabled: 0, rateLimited: 0 },
  );

  expect(res.body.length).toBe(1);
  expect(res.body[0].total).toBe(verificationsForTag.total);
  expect(res.body[0].valid).toBe(verificationsForTag.valid);
  expect(res.body[0].disabled).toBe(verificationsForTag.disabled);
  expect(res.body[0].rateLimited).toBe(verificationsForTag.rateLimited);
});
test("filter by multiple tags", { timeout: 120_000 }, async (t) => {
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

  const tags = [["a", "b"], ["a"], [], ["b", "c"]];

  const verifications = tags.flatMap((tags) =>
    generate({
      start: now - 60 * 24 * 60 * 60 * 1000,
      end: now,
      length: 100_000,
      workspaceId: h.resources.userWorkspace.id,
      keySpaceId: h.resources.userKeyAuth.id,
      keys: keys.map((k) => ({ keyId: k.keyId, identityId: k.identityId })),
      tags,
    }),
  );

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
      apiId: h.resources.userApi.id,
      tag: ["a", "b"],
    },
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const want = verifications.reduce(
    (acc, v) => {
      if (
        toStartOfHour(v.time) < start ||
        toStartOfHour(v.time) > end ||
        !v.tags.includes("a") ||
        !v.tags.includes("b")
      ) {
        return acc;
      }
      acc.total += 1;
      switch (v.outcome) {
        case "VALID": {
          acc.valid += 1;
          break;
        }
        case "DISABLED": {
          acc.disabled += 1;
          break;
        }
        case "RATE_LIMITED": {
          acc.rateLimited += 1;
          break;
        }
      }

      return acc;
    },
    { total: 0, valid: 0, disabled: 0, rateLimited: 0 },
  );

  expect(res.body.length).toBe(1);
  expect(res.body[0].total).toBe(want.total);
  expect(res.body[0].valid).toBe(want.valid);
  expect(res.body[0].disabled).toBe(want.disabled);
  expect(res.body[0].rateLimited).toBe(want.rateLimited);
});

test("filter by key", { timeout: 120_000 }, async (t) => {
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
    tags: [],
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
      apiId: h.resources.userApi.id,
      keyId: keys[0].keyId,
    },
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const verificationsForKey = verifications.reduce(
    (acc, v) => {
      if (
        toStartOfHour(v.time) < start ||
        toStartOfHour(v.time) > end ||
        v.key_id !== keys[0].keyId
      ) {
        return acc;
      }
      acc.total += 1;
      switch (v.outcome) {
        case "VALID": {
          acc.valid += 1;
          break;
        }
        case "DISABLED": {
          acc.disabled += 1;
          break;
        }
        case "RATE_LIMITED": {
          acc.rateLimited += 1;
          break;
        }
      }

      return acc;
    },
    { total: 0, valid: 0, disabled: 0, rateLimited: 0 },
  );

  expect(res.body.length).toBe(1);
  expect(res.body[0].total).toBe(verificationsForKey.total);
  expect(res.body[0].valid).toBe(verificationsForKey.valid);
  expect(res.body[0].disabled).toBe(verificationsForKey.disabled);
  expect(res.body[0].rateLimited).toBe(verificationsForKey.rateLimited);
});
test("filter by multiple keys", { timeout: 120_000 }, async (t) => {
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
    tags: [],
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
      apiId: h.resources.userApi.id,
      keyId: [keys[0].keyId, keys[1].keyId],
    },
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const want = verifications.reduce(
    (acc, v) => {
      if (
        toStartOfHour(v.time) < start ||
        toStartOfHour(v.time) > end ||
        (v.key_id !== keys[0].keyId && v.key_id !== keys[1].keyId)
      ) {
        return acc;
      }
      acc.total += 1;
      switch (v.outcome) {
        case "VALID": {
          acc.valid += 1;
          break;
        }
        case "DISABLED": {
          acc.disabled += 1;
          break;
        }
        case "RATE_LIMITED": {
          acc.rateLimited += 1;
          break;
        }
      }

      return acc;
    },
    { total: 0, valid: 0, disabled: 0, rateLimited: 0 },
  );

  expect(res.body.length).toBe(1);
  expect(res.body[0].total).toBe(want.total);
  expect(res.body[0].valid).toBe(want.valid);
  expect(res.body[0].disabled).toBe(want.disabled);
  expect(res.body[0].rateLimited).toBe(want.rateLimited);
});

test("grouping by tags", { timeout: 120_000 }, async (t) => {
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

  const tags = [["a", "b"], ["a"], [], ["b", "c"]];

  const verifications = tags.flatMap((tags) =>
    generate({
      start: now - 60 * 24 * 60 * 60 * 1000,
      end: now,
      length: 100_000,
      workspaceId: h.resources.userWorkspace.id,
      keySpaceId: h.resources.userKeyAuth.id,
      keys: keys.map((k) => ({ keyId: k.keyId, identityId: k.identityId })),
      tags,
    }),
  );

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
      apiId: h.resources.userApi.id,
      groupBy: "tags",
    },
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  expect(res.body.length).toBe(tags.length);
});

test("breakdown by tag", { timeout: 120_000 }, async (t) => {
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

  const tags = [["a", "b"], ["a"], [], ["b", "c"]];

  const verifications = tags.flatMap((tags) =>
    generate({
      start: now - 60 * 24 * 60 * 60 * 1000,
      end: now,
      length: 100_000,
      workspaceId: h.resources.userWorkspace.id,
      keySpaceId: h.resources.userKeyAuth.id,
      keys: keys.map((k) => ({ keyId: k.keyId, identityId: k.identityId })),
      tags,
    }),
  );

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
      apiId: h.resources.userApi.id,
      groupBy: "tag",
    },
    headers: {
      Authorization: `Bearer ${root.key}`,
    },
  });

  expect(res.status, `expected 200, received: ${JSON.stringify(res, null, 2)}`).toBe(200);

  const want = verifications.reduce(
    (acc, v) => {
      if (toStartOfHour(v.time) < start || toStartOfHour(v.time) > end) {
        return acc;
      }

      for (const tag of v.tags) {
        if (!acc[tag]) {
          acc[tag] = { total: 0, valid: 0, disabled: 0, rateLimited: 0 };
        }

        acc[tag].total += 1;
        switch (v.outcome) {
          case "VALID": {
            acc[tag].valid += 1;
            break;
          }
          case "DISABLED": {
            acc[tag].disabled += 1;
            break;
          }
          case "RATE_LIMITED": {
            acc[tag].rateLimited += 1;
            break;
          }
        }
      }

      return acc;
    },
    {} as Record<string, { total: 0; valid: 0; disabled: 0; rateLimited: 0 }>,
  );

  for (const row of res.body) {
    expect(row.total).toEqual(want[row.tag!].total);
    expect(row.valid).toEqual(want[row.tag!].valid);
    expect(row.disabled).toEqual(want[row.tag!].disabled);
    expect(row.rateLimited).toEqual(want[row.tag!].rateLimited);
  }
});

/**
 * Generate a number of key verification events to seed clickhouse
 * @param opts.start - Start timestamp in milliseconds
 * @param opts.end - End timestamp in milliseconds
 * @param opts.length - Number of events to generate
 * @param opts.workspaceId - Workspace identifier
 * @param opts.keySpaceId - Key space identifier
 * @param opts.keys - Array of key configurations
 * @param opts.tags - Optional array of tags
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

/**
 * Truncates the timestamp to the start of the current hour
 */
function toStartOfHour(unixmilli: number): number {
  return Math.floor(unixmilli / 60 / 60 / 1000) * 60 * 60 * 1000;
}
