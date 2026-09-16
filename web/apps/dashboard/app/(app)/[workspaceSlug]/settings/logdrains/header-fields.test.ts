import { describe, expect, it } from "vitest";
import { toHeaderRecord } from "./header-fields";

describe("toHeaderRecord", () => {
  it("drops the blank rows the form keeps for editing", () => {
    expect(toHeaderRecord([{ name: "", value: "" }])).toEqual({});
  });

  it("trims names and keeps values verbatim", () => {
    expect(
      toHeaderRecord([
        { name: " Authorization ", value: "Bearer token" },
        { name: "X-Source", value: "unkey" },
      ]),
    ).toEqual({
      Authorization: "Bearer token",
      "X-Source": "unkey",
    });
  });
});
