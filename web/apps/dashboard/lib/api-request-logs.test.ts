import { describe, expect, it } from "vitest";
import { toStatusFilter } from "./api-request-logs";

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
