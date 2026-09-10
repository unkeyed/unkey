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
  test("no filters yields an empty input", () => {
    expect(buildDeploymentListInput([], environments)).toEqual({
      input: {},
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
      [filter("status", "deploying"), filter("status", "skipped")],
      environments,
    );
    expect(input.statuses).toEqual([
      "starting",
      "building",
      "deploying",
      "network",
      "finalizing",
      "cancelled",
      "skipped",
    ]);
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
