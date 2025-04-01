import { BenchmarkHarness } from "@/pkg/testutil/benchmark-harness";
import type { V1KeysCreateKeyRequest, V1KeysCreateKeyResponse } from "@/routes/v1_keys_createKey";
import { describe, expect, test } from "vitest";

describe("fresh key per region", () => {
  describe.each<{
    name: string;
    keys: number;
    testsPerKey: number;
    threshold: {
      p50: number;
      p90: number;
      p99: number;
    };
  }>([
    {
      name: "cold",
      keys: 100,
      testsPerKey: 2,
      threshold: {
        p50: 200,
        p90: 250,
        p99: 300,
      },
    },
    {
      name: "mid",
      keys: 10,
      testsPerKey: 10,
      threshold: {
        p50: 50,
        p90: 100,
        p99: 150,
      },
    },
    {
      name: "hot",
      keys: 10,
      testsPerKey: 100,
      threshold: {
        p50: 50,
        p90: 100,
        p99: 200,
      },
    },
    {
      name: "the floor is lava",
      keys: 50,
      testsPerKey: 500,
      threshold: {
        p50: 30,
        p90: 50,
        p99: 75,
      },
    },
  ])("$name", ({ keys, testsPerKey, threshold }) => {
    describe.each([
      "arn",
      "bom",
      "cdg",
      "cle",
      "cpt",
      "dub",
      "fra",
      "gru",
      "hkg",
      "hnd",
      "iad",
      "icn",
      "kix",
      "lhr",
      "pdx",
      "sfo",
      "sin",
      "syd",
    ])("%s", (region) => {
      test(
        "latencies should be below thresholds",
        async (t) => {
          const h = await BenchmarkHarness.init(t);
          const checks = await createAndTestKeys(h, {
            keys,
            testsPerKey,
            region,
          });
          expect(checks.every(({ status }) => status === 200)).toBe(true);
          const { p50, p90, p99 } = aggregateLatencies(checks.map(({ latency }) => latency));

          expect(p50, "latency p50 is too high").toBeLessThanOrEqual(threshold.p50);
          expect(p90, "latency p90 is too high").toBeLessThanOrEqual(threshold.p90);
          expect(p99, "latency p99 is too high").toBeLessThanOrEqual(threshold.p99);
        },
        {
          timeout: 30_000,
        },
      );
    });
  });
});

function aggregateLatencies(latencies: number[]): {
  min: number;
  p50: number;
  p90: number;
  p99: number;
  max: number;
} {
  return {
    min: Math.min(...latencies),
    p50: percentile(0.5, latencies),
    p90: percentile(0.9, latencies),
    p99: percentile(0.99, latencies),
    max: Math.max(...latencies),
  };
}

function percentile(p: number, values: number[]): number {
  const sorted = values.slice().sort((a, b) => a - b);
  const index = Math.ceil(p * sorted.length) - 1;

  return sorted[index];
}

async function createAndTestKeys(
  h: BenchmarkHarness,
  opts: {
    keys: number;
    testsPerKey: number;
    region: string;
  },
): Promise<
  {
    status: number;
    latency: number;
  }[]
> {
  const { key: rootKey } = await h.createRootKey(["*", `api.${h.resources.userApi.id}.create_key`]);
  const results = await Promise.all(
    new Array(opts.keys).fill(null).map(async () => {
      const key = await h.post<V1KeysCreateKeyRequest, V1KeysCreateKeyResponse>({
        url: `${h.env.UNKEY_BASE_URL}/v1/keys.createKey`,
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${rootKey}`,
        },
        body: {
          apiId: h.resources.userApi.id,
          ratelimit: {
            async: true,
            limit: 1000,
            refillInterval: 1000,
            refillRate: 1000,
            duration: 1000,
          },
        },
      });
      expect(key.status).toEqual(200);
      expect(key.body.key).toBeDefined();
      expect(key.body.keyId).toBeDefined();
      const res = await fetch(`${h.env.PLANETFALL_URL}/api/check/${opts.region}`, {
        method: "POST",
        headers: {
          "Content-Type": "application/json",
          Authorization: `Bearer ${h.env.PLANETFALL_API_KEY}`,
        },
        body: JSON.stringify({
          method: "POST",
          url: `${h.env.UNKEY_BASE_URL}/v1/keys.verifyKey`,
          headers: { "Content-Type": "application/json" },
          body: JSON.stringify({ key: key.body.key }),
          n: opts.testsPerKey,
        }),
      });

      const { checks } = (await res.json()) as {
        checks: {
          status: number;
          latency: number;
        }[];
      };
      expect(checks.length).toBe(opts.testsPerKey);

      return checks;
    }),
  );
  return results.flat();
}
