import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";
import { DAY_IN_MS, HOUR_IN_MS, MONTH_IN_MS, QUARTER_IN_MS, WEEK_IN_MS } from "./constants";
import {
  type TimeseriesConfig,
  type TimeseriesGranularity,
  getTimeseriesGranularity,
} from "./granularity";

describe("getTimeseriesGranularity", () => {
  const originalDateNow = Date.now;
  const FIXED_NOW = 1640995200000; // 2022-01-01T00:00:00.000Z

  beforeEach(() => {
    vi.spyOn(Date, "now").mockImplementation(() => FIXED_NOW);
  });

  afterEach(() => {
    vi.restoreAllMocks();
    Date.now = originalDateNow;
  });

  const getTime = (offset: number) => FIXED_NOW - offset;

  describe("Default parameters (null startTime and endTime)", () => {
    it("should return correct defaults for forRegular context", () => {
      const result = getTimeseriesGranularity("forRegular", null, null);

      expect(result).toEqual({
        granularity: "perMinute",
        startTime: FIXED_NOW - HOUR_IN_MS,
        endTime: FIXED_NOW,
        context: "forRegular",
      });
    });

    it("should return correct defaults for forVerifications context", () => {
      const result = getTimeseriesGranularity("forVerifications", null, null);

      expect(result).toEqual({
        granularity: "perHour",
        startTime: FIXED_NOW - DAY_IN_MS,
        endTime: FIXED_NOW,
        context: "forVerifications",
      });
    });
  });

  describe("With endTime only (null startTime)", () => {
    it("should set startTime based on context for forRegular", () => {
      const endTime = FIXED_NOW;
      const result = getTimeseriesGranularity("forRegular", null, endTime);

      expect(result.startTime).toBe(endTime - HOUR_IN_MS);
      expect(result.endTime).toBe(endTime);
    });

    it("should set startTime based on context for forVerifications", () => {
      const endTime = FIXED_NOW;
      const result = getTimeseriesGranularity("forVerifications", null, endTime);

      expect(result.startTime).toBe(endTime - DAY_IN_MS);
      expect(result.endTime).toBe(endTime);
    });
  });

  describe("With startTime only (null endTime)", () => {
    it("should use current time as endTime for forRegular", () => {
      const startTime = FIXED_NOW - HOUR_IN_MS * 2;
      const result = getTimeseriesGranularity("forRegular", startTime, null);

      expect(result.startTime).toBe(startTime);
      expect(result.endTime).toBe(FIXED_NOW);
    });

    it("should use current time as endTime for forVerifications", () => {
      const startTime = FIXED_NOW - DAY_IN_MS * 2;
      const result = getTimeseriesGranularity("forVerifications", startTime, null);

      expect(result.startTime).toBe(startTime);
      expect(result.endTime).toBe(FIXED_NOW);
    });
  });

  describe("Unified granularity selection logic", () => {
    const testCases = [
      {
        name: "should use perMinute for timeRange < 1 hour",
        startTime: getTime(HOUR_IN_MS * 0.5),
        expectedGranularity: "perMinute",
      },
      {
        name: "should use perMinute for timeRange >= 1 hour & < 2 hours",
        startTime: getTime(HOUR_IN_MS * 1.5),
        expectedGranularity: "perMinute",
      },
      {
        name: "should use per5Minutes for timeRange >= 2 hours & < 4 hours",
        startTime: getTime(HOUR_IN_MS * 3),
        expectedGranularity: "per5Minutes",
      },
      {
        name: "should use per15Minutes for timeRange >= 4 hours & < 12 hours",
        startTime: getTime(HOUR_IN_MS * 8),
        expectedGranularity: "per15Minutes",
      },
      {
        name: "should use per30Minutes for timeRange >= 12 hours & < 1 day",
        startTime: getTime(HOUR_IN_MS * 18),
        expectedGranularity: "per30Minutes",
      },
      {
        name: "should use per2Hours for timeRange >= 1 day & < 3 days",
        startTime: getTime(DAY_IN_MS * 2),
        expectedGranularity: "per2Hours",
      },
      {
        name: "should use perDay for timeRange >= 3 days & < 7 days",
        startTime: getTime(DAY_IN_MS * 5),
        expectedGranularity: "perDay",
      },
      {
        name: "should use perDay for timeRange >= 7 days & < 30 days",
        startTime: getTime(DAY_IN_MS * 20),
        expectedGranularity: "perDay",
      },
      {
        name: "should use perWeek for timeRange >= 30 days & < 90 days",
        startTime: getTime(DAY_IN_MS * 60),
        expectedGranularity: "perWeek",
      },
      {
        name: "should use perMonth for timeRange >= 90 days",
        startTime: getTime(DAY_IN_MS * 120),
        expectedGranularity: "perMonth",
      },
    ];

    testCases.forEach((testCase) => {
      it(testCase.name, () => {
        const resultRegular = getTimeseriesGranularity("forRegular", testCase.startTime, FIXED_NOW);
        const resultVerifications = getTimeseriesGranularity(
          "forVerifications",
          testCase.startTime,
          FIXED_NOW,
        );

        // Both contexts should use the same granularity logic now
        expect(resultRegular.granularity).toBe(testCase.expectedGranularity);
        expect(resultVerifications.granularity).toBe(testCase.expectedGranularity);
      });
    });
  });

  describe("Boundary testing", () => {
    it("should handle edge case at exactly 1 hour boundary", () => {
      const result = getTimeseriesGranularity("forRegular", FIXED_NOW - HOUR_IN_MS, FIXED_NOW);
      expect(result.granularity).toBe("perMinute");
    });

    it("should handle edge case at exactly 2 hours boundary", () => {
      const result = getTimeseriesGranularity("forRegular", FIXED_NOW - HOUR_IN_MS * 2, FIXED_NOW);
      expect(result.granularity).toBe("per5Minutes");
    });

    it("should handle edge case at exactly 4 hours boundary", () => {
      const result = getTimeseriesGranularity("forRegular", FIXED_NOW - HOUR_IN_MS * 4, FIXED_NOW);
      expect(result.granularity).toBe("per15Minutes");
    });

    it("should handle edge case at exactly 12 hours boundary", () => {
      const result = getTimeseriesGranularity("forRegular", FIXED_NOW - HOUR_IN_MS * 12, FIXED_NOW);
      expect(result.granularity).toBe("per30Minutes");
    });

    it("should handle edge case at exactly 1 day boundary", () => {
      const result = getTimeseriesGranularity("forRegular", FIXED_NOW - DAY_IN_MS, FIXED_NOW);
      expect(result.granularity).toBe("per2Hours");
    });

    it("should handle edge case at exactly 3 days boundary", () => {
      const result = getTimeseriesGranularity("forRegular", FIXED_NOW - DAY_IN_MS * 3, FIXED_NOW);
      expect(result.granularity).toBe("per2Hours");
    });

    it("should handle edge case at exactly 7 days boundary", () => {
      const result = getTimeseriesGranularity("forRegular", FIXED_NOW - WEEK_IN_MS, FIXED_NOW);
      expect(result.granularity).toBe("perDay");
    });

    it("should handle edge case at exactly 30 days boundary", () => {
      const result = getTimeseriesGranularity("forRegular", FIXED_NOW - MONTH_IN_MS, FIXED_NOW);
      expect(result.granularity).toBe("perWeek");
    });

    it("should handle edge case at exactly 90 days boundary", () => {
      const result = getTimeseriesGranularity("forRegular", FIXED_NOW - QUARTER_IN_MS, FIXED_NOW);
      expect(result.granularity).toBe("perMonth");
    });
  });

  describe("Type compatibility tests", () => {
    it("should properly type the return for forRegular context", () => {
      const result: TimeseriesConfig<"forRegular"> = getTimeseriesGranularity(
        "forRegular",
        null,
        null,
      );
      expect(result.context).toBe("forRegular");

      const validGranularities: TimeseriesGranularity[] = [
        "perMinute",
        "per5Minutes",
        "per15Minutes",
        "per30Minutes",
        "perHour",
        "per2Hours",
        "per4Hours",
        "per6Hours",
        "per12Hours",
        "perDay",
        "per3Days",
        "perWeek",
        "perMonth",
        "perQuarter",
      ];

      expect(validGranularities.includes(result.granularity)).toBeTruthy();
    });

    it("should properly type the return for forVerifications context", () => {
      const result: TimeseriesConfig<"forVerifications"> = getTimeseriesGranularity(
        "forVerifications",
        null,
        null,
      );
      expect(result.context).toBe("forVerifications");

      const validGranularities: TimeseriesGranularity[] = [
        "perMinute",
        "per5Minutes",
        "per15Minutes",
        "per30Minutes",
        "perHour",
        "per2Hours",
        "per4Hours",
        "per6Hours",
        "per12Hours",
        "perDay",
        "per3Days",
        "perWeek",
        "perMonth",
        "perQuarter",
      ];

      expect(validGranularities.includes(result.granularity)).toBeTruthy();
    });
  });

  describe("Real-world use cases", () => {
    it("should handle a 1-hour dashboard view correctly", () => {
      const oneHourAgo = FIXED_NOW - HOUR_IN_MS;
      const result = getTimeseriesGranularity("forRegular", oneHourAgo, FIXED_NOW);

      expect(result.granularity).toBe("perMinute");
      expect(result.startTime).toBe(oneHourAgo);
      expect(result.endTime).toBe(FIXED_NOW);
    });

    it("should handle a 24-hour dashboard view correctly", () => {
      const oneDayAgo = FIXED_NOW - DAY_IN_MS;
      const result = getTimeseriesGranularity("forRegular", oneDayAgo, FIXED_NOW);

      expect(result.granularity).toBe("per2Hours");
      expect(result.startTime).toBe(oneDayAgo);
      expect(result.endTime).toBe(FIXED_NOW);
    });

    it("should handle a 1-week dashboard view correctly", () => {
      const oneWeekAgo = FIXED_NOW - WEEK_IN_MS;
      const result = getTimeseriesGranularity("forRegular", oneWeekAgo, FIXED_NOW);

      expect(result.granularity).toBe("perDay");
      expect(result.startTime).toBe(oneWeekAgo);
      expect(result.endTime).toBe(FIXED_NOW);
    });

    it("should handle a 30-day verification dashboard view correctly", () => {
      const thirtyDaysAgo = FIXED_NOW - MONTH_IN_MS;
      const result = getTimeseriesGranularity("forVerifications", thirtyDaysAgo, FIXED_NOW);

      expect(result.granularity).toBe("perWeek");
      expect(result.startTime).toBe(thirtyDaysAgo);
      expect(result.endTime).toBe(FIXED_NOW);
    });

    it("should handle a quarterly verification dashboard view correctly", () => {
      const threeMonthsAgo = FIXED_NOW - QUARTER_IN_MS;
      const result = getTimeseriesGranularity("forVerifications", threeMonthsAgo, FIXED_NOW);

      expect(result.granularity).toBe("perMonth");
      expect(result.startTime).toBe(threeMonthsAgo);
      expect(result.endTime).toBe(FIXED_NOW);
    });

    it("should handle a 6-month dashboard view correctly", () => {
      const sixMonthsAgo = FIXED_NOW - QUARTER_IN_MS * 2;
      const result = getTimeseriesGranularity("forRegular", sixMonthsAgo, FIXED_NOW);

      expect(result.granularity).toBe("perMonth");
      expect(result.startTime).toBe(sixMonthsAgo);
      expect(result.endTime).toBe(FIXED_NOW);
    });

    it("should handle a 3-hour dashboard view correctly", () => {
      const threeHoursAgo = FIXED_NOW - HOUR_IN_MS * 3;
      const result = getTimeseriesGranularity("forRegular", threeHoursAgo, FIXED_NOW);

      expect(result.granularity).toBe("per5Minutes");
      expect(result.startTime).toBe(threeHoursAgo);
      expect(result.endTime).toBe(FIXED_NOW);
    });
  });

  describe("Edge cases and error handling", () => {
    it("should handle zero time range", () => {
      const result = getTimeseriesGranularity("forRegular", FIXED_NOW, FIXED_NOW);

      expect(result.granularity).toBe("perMinute");
      expect(result.startTime).toBe(FIXED_NOW);
      expect(result.endTime).toBe(FIXED_NOW);
    });

    it("should handle very small time ranges", () => {
      const result = getTimeseriesGranularity(
        "forRegular",
        FIXED_NOW - 1000, // 1 second
        FIXED_NOW,
      );

      expect(result.granularity).toBe("perMinute");
    });

    it("should handle negative time ranges gracefully", () => {
      const result = getTimeseriesGranularity(
        "forRegular",
        FIXED_NOW + HOUR_IN_MS, // future start time
        FIXED_NOW,
      );

      expect(result.granularity).toBe("perMinute");
      expect(result.startTime).toBe(FIXED_NOW + HOUR_IN_MS);
      expect(result.endTime).toBe(FIXED_NOW);
    });
  });
});
