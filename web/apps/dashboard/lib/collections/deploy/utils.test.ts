import { describe, expect, it } from "vitest";
import { extractStringsFilter } from "./utils";

describe("extractStringsFilter", () => {
  it("reads an eq filter as one value", () => {
    expect(
      extractStringsFilter([{ field: ["appId"], operator: "eq", value: "app_a" }], "appId"),
    ).toEqual(["app_a"]);
  });

  it("reads the string values of an in filter", () => {
    expect(
      extractStringsFilter(
        [{ field: ["appId"], operator: "in", value: ["app_a", 7, "KEBAP"] }],
        "appId",
      ),
    ).toEqual(["app_a", "KEBAP"]);
  });

  it("ignores other fields and returns no values without a match", () => {
    expect(
      extractStringsFilter([{ field: ["projectId"], operator: "eq", value: "proj_1" }], "appId"),
    ).toEqual([]);
  });
});
