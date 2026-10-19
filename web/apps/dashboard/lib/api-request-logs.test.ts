import { HOUR_IN_MS } from "@/lib/trpc/routers/utils/constants";
import { describe, expect, it } from "vitest";
import { toApiRequestsFilters, toStatusFilter } from "./api-request-logs";
import {
  type LogsFilterField,
  type LogsFilterOperator,
  type LogsFilterValue,
  logsFilterFieldConfig,
  logsSearchFieldEnum,
} from "./schemas/logs.filter.schema";

describe("toStatusFilter", () => {
  it("turns families into the one range that covers them", () => {
    expect(toStatusFilter([200])).toEqual({ gte: 200, lt: 300 });
    expect(toStatusFilter([400, 500])).toEqual({ gte: 400, lt: 600 });
    expect(toStatusFilter([404, 429])).toEqual({ is: [404, 429] });
    expect(toStatusFilter([])).toBeUndefined();
  });

  it("matches every status the old family filter matched", () => {
    for (const family of [200, 400, 500]) {
      const filter = toStatusFilter([family]);
      for (let status = 100; status < 600; status++) {
        const matches = status >= (filter?.gte ?? 0) && status < (filter?.lt ?? 0);
        expect(matches, `${family} vs ${status}`).toBe(status >= family && status < family + 100);
      }
    }
  });
});

describe("toApiRequestsFilters", () => {
  const window = { startTime: 1_000_000_000_000, endTime: 1_000_000_000_000 + 12 * HOUR_IN_MS };
  const filter = (
    field: LogsFilterField,
    operator: LogsFilterOperator,
    value: string | number,
  ): LogsFilterValue => ({ id: crypto.randomUUID(), field, operator, value });

  it("builds the API filters of the /logs page", () => {
    expect(
      toApiRequestsFilters(
        [
          filter("status", "is", 400),
          filter("methods", "is", "POST"),
          filter("paths", "startsWith", "/v2/keys."),
          filter("paths", "contains", "KEBAP"),
          filter("host", "is", "api.unkey.com"),
          filter("since", "is", "1h"),
        ],
        window,
      ),
    ).toEqual({
      time: { gte: window.endTime - HOUR_IN_MS, lte: window.endTime },
      path: { startsWith: ["/v2/keys."], contains: ["KEBAP"] },
      host: { is: ["api.unkey.com"] },
      method: { is: ["POST"] },
      requestId: undefined,
      status: { gte: 400, lt: 500 },
    });
  });

  it("sends every filter AI search can emit to the API", () => {
    const empty = toApiRequestsFilters([], window);
    const sample: Record<LogsFilterField, string | number> = {
      host: "api.unkey.com",
      requestId: "req_KEBAP",
      methods: "GET",
      paths: "/v2/KEBAP",
      status: 500,
      appId: "app_KEBAP",
      deploymentId: "d_KEBAP",
      environmentId: "env_KEBAP",
      startTime: window.startTime + 1,
      endTime: window.endTime - 1,
      since: "1h",
    };
    for (const field of logsSearchFieldEnum.options) {
      for (const operator of logsFilterFieldConfig[field].operators) {
        expect(
          toApiRequestsFilters([filter(field, operator, sample[field])], window),
          `${field} ${operator}`,
        ).not.toEqual(empty);
      }
    }
  });
});
