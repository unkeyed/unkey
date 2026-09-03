import { describe, expect, it } from "vitest";
import {
  alertMetricLabel,
  alertSeriesMetricLabel,
  formatAlertAxisValue,
  formatAlertDistance,
  formatAlertSeriesValue,
  formatAlertValue,
  formatBaselineMultiple,
  formatRequestsDropChange,
  seriesMetricForAlert,
} from "./format";

describe("alert metric formatting", () => {
  it("uses human-readable labels", () => {
    expect(alertMetricLabel("error_5xx")).toBe("5xx errors");
    expect(alertMetricLabel("requests_drop")).toBe("Traffic drop");
    expect(alertMetricLabel("oom_killed")).toBe("Out of memory");
  });

  it("maps alert-only metrics to their shared chart series", () => {
    expect(seriesMetricForAlert("requests_drop")).toBe("requests");
    expect(seriesMetricForAlert("oom_killed")).toBe("health");
    expect(seriesMetricForAlert("crash_loop")).toBe("health");
    expect(seriesMetricForAlert("egress_bytes")).toBe("egress_bytes");
    expect(alertSeriesMetricLabel("health")).toBe("Health");
    expect(formatAlertSeriesValue("health", 3)).toBe("3");
  });

  it("formats each unit family", () => {
    expect(formatAlertValue("egress_bytes", 3_250_000)).toBe("3.1 MB");
    expect(formatAlertValue("cpu_seconds", 12.345)).toBe("12.35 s");
    expect(formatAlertValue("memory_utilization", 0.873)).toBe("87.3%");
    expect(formatAlertValue("error_5xx", 0.032)).toBe("3.2%");
    expect(formatAlertValue("requests", 1_234)).toBe("1,234");
  });

  it("uses compact values on chart axes", () => {
    expect(formatAlertAxisValue("requests", 12_400)).toBe("12.4K");
  });

  it("formats a plain-language baseline comparison", () => {
    expect(formatBaselineMultiple(3.8, 0.4)).toBe("9.5× baseline");
    expect(formatBaselineMultiple(42, 1)).toBe("42× baseline");
    expect(formatBaselineMultiple(1, 11)).toBe("below baseline");
    expect(formatBaselineMultiple(2, 0)).toBe("no prior traffic");
  });

  it("shows fixed limits and error-specific empty baselines", () => {
    expect(formatAlertDistance("memory_utilization", 0.94, 0)).toBe("avg 94% · limit 90%");
    expect(formatAlertDistance("oom_killed", 3, 0)).toBe("3 events · limit 1");
    expect(formatAlertDistance("crash_loop", 4, 0)).toBe("4 events · limit 1");
    expect(formatAlertDistance("error_5xx", 0.032, 0.004)).toBe("8.0× baseline");
    expect(formatAlertDistance("error_5xx", 0.032, 0)).toBe("no prior errors");
  });

  it("formats traffic drops against the recent median without infinity", () => {
    expect(formatRequestsDropChange(1, 12)).toBe("−92%");
    expect(formatAlertDistance("requests_drop", 1, 12)).toBe("−92%");
    expect(formatRequestsDropChange(1, 0)).toBe("No recent traffic");
  });
});
