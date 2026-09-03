import { describe, expect, it } from "vitest";
import {
  alertMetricLabel,
  formatAlertAxisValue,
  formatAlertDistance,
  formatAlertValue,
  formatSigma,
} from "./format";

describe("alert metric formatting", () => {
  it("uses human-readable labels", () => {
    expect(alertMetricLabel("error_5xx")).toBe("5xx errors");
    expect(alertMetricLabel("requests_drop")).toBe("Traffic drop");
    expect(alertMetricLabel("oom_killed")).toBe("Out of memory");
  });

  it("formats each unit family", () => {
    expect(formatAlertValue("egress_bytes", 3_250_000)).toBe("3.1 MB");
    expect(formatAlertValue("cpu_seconds", 12.345)).toBe("12.35 s");
    expect(formatAlertValue("memory_utilization", 0.873)).toBe("87.3%");
    expect(formatAlertValue("requests", 1_234)).toBe("1,234");
  });

  it("uses compact values on chart axes", () => {
    expect(formatAlertAxisValue("requests", 12_400)).toBe("12.4K");
  });

  it("formats the distance above the baseline", () => {
    expect(formatSigma(3.8, 0.4, 0.5)).toBe("+6.8σ");
    expect(formatSigma(1, 11, 2)).toBe("-5.0σ");
    expect(formatSigma(2, 1, 0)).toBe("No variance");
  });

  it("shows fixed limits instead of sigma for threshold metrics", () => {
    expect(formatAlertDistance("memory_utilization", 0.94, 0, 0)).toBe("94% · limit 90%");
    expect(formatAlertDistance("oom_killed", 3, 0, 0)).toBe("3 events · limit 1");
    expect(formatAlertDistance("crash_loop", 4, 0, 0)).toBe("4 events · limit 1");
    expect(formatAlertDistance("error_5xx", 8, 1, 1)).toBe("+7.0σ");
  });
});
