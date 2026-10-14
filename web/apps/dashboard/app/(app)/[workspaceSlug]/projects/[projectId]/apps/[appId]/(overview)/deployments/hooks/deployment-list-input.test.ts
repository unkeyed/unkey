import { DEPLOYMENT_STATUSES } from "@/lib/collections/deploy/deployment-status";
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
  test("no filters asks for every status except skipped", () => {
    expect(buildDeploymentListInput([], environments)).toEqual({
      filter: { statuses: DEPLOYMENT_STATUSES.filter((s) => s !== "skipped"), branches: [] },
      cannotMatch: false,
    });
  });

  test("expands status groups into raw statuses", () => {
    const { filter: listFilter } = buildDeploymentListInput(
      [filter("status", "building"), filter("status", "ready")],
      environments,
    );
    expect(listFilter.statuses).toEqual([
      "starting",
      "building",
      "deploying",
      "network",
      "finalizing",
      "ready",
    ]);
  });

  test("maps the previous filter bar's status values onto their groups", () => {
    const { filter: listFilter } = buildDeploymentListInput(
      [filter("status", "deploying"), filter("status", "network"), filter("status", "pending")],
      environments,
    );
    expect(listFilter.statuses).toEqual([
      "starting",
      "building",
      "deploying",
      "network",
      "finalizing",
      "pending",
    ]);
  });

  test("sends the skipped group unchanged", () => {
    const { filter: listFilter } = buildDeploymentListInput(
      [filter("status", "skipped")],
      environments,
    );
    expect(listFilter.statuses).toEqual(["skipped"]);
  });

  test("flags a status that is not a group as unable to match", () => {
    const result = buildDeploymentListInput([filter("status", "constructor")], environments);
    expect(result.cannotMatch).toBe(true);
  });

  test("resolves a single environment slug to its id", () => {
    const { filter: listFilter, cannotMatch } = buildDeploymentListInput(
      [filter("environment", "production")],
      environments,
    );
    expect(listFilter.environmentId).toBe("env_prod");
    expect(cannotMatch).toBe(false);
  });

  test("flags an environment slug this app does not have", () => {
    const result = buildDeploymentListInput([filter("environment", "staging")], environments);
    expect(result.filter.environmentId).toBeUndefined();
    expect(result.cannotMatch).toBe(true);
  });

  test("treats both time bounds as inclusive", () => {
    const at = (start: number, end: number) =>
      buildDeploymentListInput([filter("startTime", start), filter("endTime", end)], environments)
        .cannotMatch;
    expect(at(1_000, 1_000)).toBe(false);
    expect(at(1_001, 1_000)).toBe(true);
  });

  test("passes branches and explicit time bounds through", () => {
    const { filter: listFilter } = buildDeploymentListInput(
      [filter("branch", "main"), filter("startTime", 1_000), filter("endTime", 2_000)],
      environments,
    );
    expect(listFilter).toMatchObject({ branches: ["main"], startTime: 1_000, endTime: 2_000 });
  });

  test("turns a relative window into a start time floored to the minute", () => {
    const now = 1_700_000_000_123;
    const { filter: listFilter } = buildDeploymentListInput(
      [filter("since", "1h")],
      environments,
      now,
    );
    const expected = Math.floor((now - 60 * 60 * 1000) / 60_000) * 60_000;
    expect(listFilter.startTime).toBe(expected);
  });

  test.each([
    ["1w", "2026-09-21T12:34:00Z"],
    ["7d", "2026-09-21T12:34:00Z"],
    ["1w2d3h30m", "2026-09-19T09:04:00Z"],
  ])("converts the URL relative window %s", (since, expected) => {
    const now = Date.parse("2026-09-28T12:34:56.789Z");
    const { filter: listFilter } = buildDeploymentListInput(
      [filter("since", since)],
      environments,
      now,
    );
    expect(listFilter.startTime).toBe(Date.parse(expected));
  });

  test("keeps the later of an explicit start and a relative window", () => {
    const now = 1_700_000_000_000;
    const { filter: listFilter } = buildDeploymentListInput(
      [filter("since", "1h"), filter("startTime", now)],
      environments,
      now,
    );
    expect(listFilter.startTime).toBe(now);
  });
});
