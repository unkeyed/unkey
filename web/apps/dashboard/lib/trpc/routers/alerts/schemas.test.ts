import { describe, expect, it } from "vitest";
import { alertDeploymentsInput, alertSeriesInput, listAlertsInput } from "./schemas";

describe("alert inputs", () => {
  it("applies list defaults", () => {
    expect(listAlertsInput.parse({})).toEqual({ limit: 50 });
  });

  it("accepts traffic drop alerts", () => {
    expect(listAlertsInput.parse({ metric: "requests_drop" })).toEqual({
      metric: "requests_drop",
      limit: 50,
    });
  });

  it("accepts an app environment and time range", () => {
    expect(
      listAlertsInput.parse({
        appId: "app_1",
        environmentId: "env_1",
        startMs: 100,
        endMs: 200,
      }),
    ).toMatchObject({ appId: "app_1", environmentId: "env_1", startMs: 100, endMs: 200 });
  });

  it.each([
    { metric: "latency" },
    { limit: 0 },
    { limit: 101 },
    { cursor: "" },
    { startMs: 200, endMs: 100 },
  ])("rejects invalid list input", (input) => {
    expect(listAlertsInput.safeParse(input).success).toBe(false);
  });

  it("validates metric series input", () => {
    expect(
      alertSeriesInput.parse({
        appId: "app_1",
        environmentId: "env_1",
        metric: "health",
        startMs: 100,
        endMs: 200,
      }),
    ).toMatchObject({ metric: "health", resolution: "5m" });
    expect(
      alertSeriesInput.safeParse({
        appId: "app_1",
        environmentId: "env_1",
        metric: "requests_drop",
        startMs: 100,
        endMs: 200,
      }).success,
    ).toBe(false);
  });

  it("validates deployment marker ranges", () => {
    expect(
      alertDeploymentsInput.safeParse({
        appId: "app_1",
        environmentId: "env_1",
        startMs: 200,
        endMs: 100,
      }).success,
    ).toBe(false);
  });
});
