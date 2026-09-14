import { describe, expect, it } from "vitest";
import { buildRequestDetailsQueryParams } from "./use-fetch-request-details";

describe("buildRequestDetailsQueryParams", () => {
  const time = 1_757_872_504_000;

  it("brackets the window around the row's own time", () => {
    const params = buildRequestDetailsQueryParams({ requestId: "req_1", time });

    expect(params.startTime).toBeLessThan(time);
    expect(params.endTime).toBeGreaterThan(time);
    expect(params.requestId).toEqual({ filters: [{ operator: "is", value: "req_1" }] });
    expect(params.limit).toBe(1);
  });

  it("covers a row stamped after the page's query clock", () => {
    // The regression: live-tailed rows arrive with a time past the frozen
    // query clock, and an endTime taken from that clock excluded them forever.
    const staleQueryClock = time - 5 * 60_000;
    const params = buildRequestDetailsQueryParams({ requestId: "req_1", time });

    expect(params.endTime).toBeGreaterThan(staleQueryClock);
    expect(params.endTime).toBeGreaterThanOrEqual(time);
  });

  it("reaches back far enough for a request that started before its verification", () => {
    // api_requests rows are stamped at request start, verification rows
    // mid-handler; the API caps a request at a one minute timeout.
    const params = buildRequestDetailsQueryParams({ requestId: "req_1", time });

    expect(params.startTime).toBeLessThanOrEqual(time - 60_000);
  });

  it("never asks for a negative start time", () => {
    const params = buildRequestDetailsQueryParams({ requestId: "req_1", time: 0 });

    expect(params.startTime).toBe(0);
  });

  it("drops the request filter when there is no selected row", () => {
    expect(buildRequestDetailsQueryParams({}).requestId).toBeNull();
  });
});
