import { describe, expect, it } from "vitest";
import { describePermission } from "./describe-permission";

describe("describePermission", () => {
  it("describes both legacy permissions and v2 permission URNs", () => {
    expect(describePermission("api.*.read_api")).toBe("Read API");
    expect(describePermission("unkey:v1:ws_1:apis/*#read_api")).toBe("Read API");
  });
});
