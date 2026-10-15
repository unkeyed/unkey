import { describe, expect, it } from "vitest";
import { withOperator } from "./kinds";

describe("withOperator", () => {
  it.each(["present", "absent"] as const)(
    "turns a header into a %s check and drops its value",
    (operator) => {
      expect(
        withOperator({ type: "header", name: "X-Tenant", operator: "regex", value: "a" }, operator),
      ).toEqual({ type: "header", name: "X-Tenant", operator, value: "" });
    },
  );

  it("ignores an operator the field does not allow", () => {
    const path = { type: "path" as const, operator: "exact" as const, value: "/v1" };
    expect(withOperator(path, "present")).toEqual(path);
    expect(withOperator(path, "absent")).toEqual(path);
    const ip = { type: "remoteIp" as const, operator: "in" as const, value: "10.0.0.0/8" };
    expect(withOperator(ip, "regex")).toEqual(ip);
  });
});
