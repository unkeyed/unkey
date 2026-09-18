// @vitest-environment node
import { SEARCH_TIMEOUT_MS } from "@/lib/search/client";
import { runSearch } from "@/lib/search/engine";
import type { FilterOutput } from "@/lib/search/spec";
import OpenAI from "openai";
import { describe, expect, it } from "vitest";
import type { GoldenQuery } from "./queries";
import { SUITES } from "./suites";

const OPENAI_KEY = process.env.OPENAI_API_KEY;
const REPEATS = Number(process.env.GOLDEN_REPEATS ?? 1);
const ONLY = process.env.GOLDEN_SUITE;
const REFERENCE_MS = Date.UTC(2026, 0, 23, 15, 0, 0);

const MIN_EXACT_PCT = 80;
const MIN_SUITE_EXACT_PCT = 60;
const MIN_PRECISION_PCT = 90;
const MIN_RECALL_PCT = 90;

function flatten(result: FilterOutput): Set<string> {
  const keys = new Set<string>();
  for (const group of result.filters) {
    for (const filter of group.filters) {
      keys.add(`${group.field}|${filter.operator}|${filter.value}`);
    }
  }
  return keys;
}

const wanted = (golden: GoldenQuery) =>
  new Set(golden.expected.map((e) => `${e.field}|${e.operator}|${e.value}`));

describe.skipIf(!OPENAI_KEY)("golden search queries", () => {
  it("reads every surface's filters correctly enough to ship", { timeout: 3_600_000 }, async () => {
    const openai = new OpenAI({ apiKey: OPENAI_KEY, timeout: SEARCH_TIMEOUT_MS });
    const suites = ONLY ? SUITES.filter((s) => s.name === ONLY) : SUITES;

    let runs = 0;
    let exact = 0;
    let hits = 0;
    let predicted = 0;
    let expected = 0;
    const rows = [];
    const mismatches: string[] = [];

    for (const suite of suites) {
      let suiteExact = 0;
      let suiteRuns = 0;
      const latencies: number[] = [];

      for (let repeat = 0; repeat < REPEATS; repeat++) {
        for (const golden of suite.queries) {
          const start = performance.now();
          const run = await runSearch(suite.spec, openai, golden.query, REFERENCE_MS);
          latencies.push(performance.now() - start);

          const got = flatten(run.result);
          const want = wanted(golden);
          runs++;
          suiteRuns++;
          predicted += got.size;
          expected += want.size;
          for (const key of got) {
            if (want.has(key)) {
              hits++;
            }
          }
          if (want.size === got.size && [...want].every((k) => got.has(k))) {
            exact++;
            suiteExact++;
          } else if (repeat === 0) {
            mismatches.push(
              `[${suite.name}] "${golden.query}"\n  want: ${[...want].sort().join(", ")}\n  got : ${[...got].sort().join(", ")}`,
            );
          }
        }
      }

      rows.push({
        suite: suite.name,
        runs: suiteRuns,
        exactPct: Number(((suiteExact / suiteRuns) * 100).toFixed(1)),
        meanMs: Math.round(latencies.reduce((a, b) => a + b, 0) / latencies.length),
      });
    }

    console.table(rows);
    if (mismatches.length > 0) {
      console.log(`\n${mismatches.length} mismatches:\n\n${mismatches.join("\n\n")}`);
    }

    for (const row of rows) {
      expect(row.exactPct, `${row.suite} exact match`).toBeGreaterThanOrEqual(MIN_SUITE_EXACT_PCT);
    }
    expect((exact / runs) * 100).toBeGreaterThanOrEqual(MIN_EXACT_PCT);
    expect((hits / predicted) * 100).toBeGreaterThanOrEqual(MIN_PRECISION_PCT);
    expect((hits / expected) * 100).toBeGreaterThanOrEqual(MIN_RECALL_PCT);
  });
});
