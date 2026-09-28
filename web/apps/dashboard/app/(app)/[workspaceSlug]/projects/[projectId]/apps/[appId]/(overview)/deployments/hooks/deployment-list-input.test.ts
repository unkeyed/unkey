import type { Environment } from "@/lib/collections/deploy/environments";
import { describe, expect, test } from "vitest";
import type { DeploymentListFilterValue } from "../filters.schema";
import { buildDeploymentListInput } from "./deployment-list-input";

const environments: Environment[] = [
  { id: "env_prod", projectId: "proj", appId: "app", slug: "production", kind: "production" },
  { id: "env_prev", projectId: "proj", appId: "app", slug: "preview", kind: "preview" },
];

const filter = (
  field: DeploymentListFilterValue["field"],
  value: string | number,
): DeploymentListFilterValue => ({ id: `${field}:${value}`, field, operator: "is", value });

describe("buildDeploymentListInput", () => {
  test("no filters limits only failures to seven days", () => {
    expect(buildDeploymentListInput([], environments, Date.UTC(2026, 8, 28, 12, 0, 35))).toEqual({
      input: { failedSince: Date.UTC(2026, 8, 21, 12) },
      cannotMatch: false,
    });
  });

  test("keeps the cutoff with default statuses, branch and environment filters", () => {
    const filters = [
      ...["blocked", "queued", "building", "failed", "ready"].map((status) =>
        filter("status", status),
      ),
      filter("branch", "main"),
      filter("environment", "production"),
    ];
    const { input } = buildDeploymentListInput(filters, environments, Date.UTC(2026, 8, 28));
    expect(input.failedSince).toBe(Date.UTC(2026, 8, 21));
    expect(input.startTime).toBeUndefined();
    expect(input.statuses).toContain("ready");
    expect(input.statuses).toContain("failed");
    expect(input.environmentIds).toEqual(["env_prod"]);
    expect(input.branches).toEqual(["main"]);
  });

  test.each([
    [filter("status", "failed")],
    [filter("status", "failed"), filter("status", "ready")],
    [filter("since", "30d")],
    [filter("startTime", 0)],
    [filter("endTime", 1_000)],
  ])("does not hide older failures for explicit status or time filters: %j", (...filters) => {
    expect(buildDeploymentListInput(filters, environments).input.failedSince).toBeUndefined();
  });

  test("show older failures removes only the failure cutoff", () => {
    const filters = [filter("branch", "main"), filter("environment", "production")];
    expect(buildDeploymentListInput(filters, environments, Date.UTC(2026, 8, 28), true)).toEqual({
      input: { branches: ["main"], environmentIds: ["env_prod"] },
      cannotMatch: false,
    });
  });

  test("expands status groups into raw statuses", () => {
    const { input } = buildDeploymentListInput(
      [filter("status", "building"), filter("status", "ready")],
      environments,
    );
    expect(input.statuses).toEqual([
      "starting",
      "building",
      "deploying",
      "network",
      "finalizing",
      "ready",
    ]);
  });

  test("maps the previous filter bar's status values onto their groups", () => {
    const { input } = buildDeploymentListInput(
      [filter("status", "deploying"), filter("status", "pending")],
      environments,
    );
    expect(input.statuses).toEqual([
      "starting",
      "building",
      "deploying",
      "network",
      "finalizing",
      "pending",
    ]);
  });

  test("keeps skipped separate from cancelled", () => {
    const { input } = buildDeploymentListInput([filter("status", "skipped")], environments);
    expect(input.statuses).toEqual(["skipped"]);
  });

  test("flags a status that is not a group as unable to match", () => {
    const result = buildDeploymentListInput([filter("status", "constructor")], environments);
    expect(result.input.statuses).toBeUndefined();
    expect(result.cannotMatch).toBe(true);
  });

  test("resolves environment slugs to ids", () => {
    const { input, cannotMatch } = buildDeploymentListInput(
      [filter("environment", "production")],
      environments,
    );
    expect(input.environmentIds).toEqual(["env_prod"]);
    expect(cannotMatch).toBe(false);
  });

  test("flags an environment slug this app does not have", () => {
    const result = buildDeploymentListInput([filter("environment", "staging")], environments);
    expect(result.input.environmentIds).toBeUndefined();
    expect(result.cannotMatch).toBe(true);
  });

  test("passes branches and explicit time bounds through", () => {
    const { input } = buildDeploymentListInput(
      [filter("branch", "main"), filter("startTime", 1_000), filter("endTime", 2_000)],
      environments,
    );
    expect(input).toEqual({ branches: ["main"], startTime: 1_000, endTime: 2_000 });
  });

  test("turns a relative window into a start time floored to the minute", () => {
    const now = 1_700_000_000_123;
    const { input } = buildDeploymentListInput([filter("since", "1h")], environments, now);
    const expected = Math.floor((now - 60 * 60 * 1000) / 60_000) * 60_000;
    expect(input.startTime).toBe(expected);
  });

  test("keeps the later of an explicit start and a relative window", () => {
    const now = 1_700_000_000_000;
    const { input } = buildDeploymentListInput(
      [filter("since", "1h"), filter("startTime", now)],
      environments,
      now,
    );
    expect(input.startTime).toBe(now);
  });
});
