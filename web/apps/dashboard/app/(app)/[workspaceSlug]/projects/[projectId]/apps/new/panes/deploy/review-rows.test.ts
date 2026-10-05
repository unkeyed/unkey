import { describe, expect, it } from "vitest";
import { reviewRows } from "./review-rows";

describe("reviewRows", () => {
  it("lists a repository app", () => {
    expect(
      reviewRows({
        source: "git",
        app: { name: "storefront", imageReference: null },
        repository: { fullName: "acme/storefront", branch: "main" },
        runtime: {
          port: 3000,
          regions: ["us-east-1", "eu-central-1"],
          size: { cpuMillicores: 250, memoryMib: 256 },
        },
        variableCount: 2,
      }),
    ).toEqual([
      { label: "Source", value: { type: "text", text: "GitHub repository" } },
      { label: "Repository", value: { type: "mono", text: "acme/storefront · main" } },
      {
        label: "App name",
        value: { type: "text", text: "storefront" },
      },
      { label: "Port", value: { type: "mono", text: "3000" } },
      { label: "Regions", value: { type: "regions", names: ["us-east-1", "eu-central-1"] } },
      { label: "Size", value: { type: "size", size: { cpuMillicores: 250, memoryMib: 256 } } },
      { label: "Environment variables", value: { type: "text", text: "2 set" } },
    ]);
  });

  it("shows loading values until the data arrives", () => {
    const rows = reviewRows({
      source: "oci",
      app: undefined,
      repository: undefined,
      runtime: undefined,
      variableCount: 0,
    });
    expect(rows.map((row) => [row.label, row.value.type])).toEqual([
      ["Source", "text"],
      ["Image", "loading"],
      ["App name", "loading"],
      ["Port", "loading"],
      ["Regions", "loading"],
      ["Size", "loading"],
      ["Environment variables", "text"],
    ]);
  });
});
