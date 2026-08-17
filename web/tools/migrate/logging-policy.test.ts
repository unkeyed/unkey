import { describe, expect, test } from "vitest";
import { reconcileLoggingPolicy } from "./logging-policy";

function isRecord(value: unknown): value is Record<string, unknown> {
  return typeof value === "object" && value !== null && !Array.isArray(value);
}

function readLoggingPolicy(blob: string | null): Record<string, unknown> {
  if (blob === null) {
    throw new Error("expected a patched blob");
  }
  const config: unknown = JSON.parse(blob);
  if (!isRecord(config) || !Array.isArray(config.policies) || !isRecord(config.policies[0])) {
    throw new Error("expected one policy");
  }
  return config.policies[0];
}

function oldBackfill(policyId: string): string {
  return JSON.stringify({
    policies: [
      {
        id: policyId,
        name: "Log everything",
        enabled: true,
        logging: {
          requestHeaders: true,
          responseHeaders: true,
          requestBody: true,
          responseBody: true,
          query: true,
        },
      },
    ],
  });
}

describe("logging policy migration", () => {
  test("production creates one policy enabled in both environments", () => {
    const production = reconcileLoggingPolicy("{}", "production#1");
    const preview = reconcileLoggingPolicy("{}", "production#2", production.policyId);

    const productionPolicy = readLoggingPolicy(production.blob);
    const previewPolicy = readLoggingPolicy(preview.blob);
    expect(productionPolicy.id).toBe(previewPolicy.id);
    expect(productionPolicy.enabled).toBe(true);
    expect(previewPolicy.enabled).toBe(true);
  });

  test("canary deduplicates backfills created with different IDs", () => {
    const production = reconcileLoggingPolicy(oldBackfill("policy_canary_production"), "canary#1");
    const preview = reconcileLoggingPolicy(
      oldBackfill("policy_canary_preview"),
      "canary#2",
      production.policyId,
    );

    expect(production.blob).toBeNull();
    expect(production.policyId).toBe("policy_canary_production");
    expect(readLoggingPolicy(preview.blob).id).toBe("policy_canary_production");

    const rerun = reconcileLoggingPolicy(preview.blob, "canary#2", production.policyId);
    expect(rerun.blob).toBeNull();
  });
});
