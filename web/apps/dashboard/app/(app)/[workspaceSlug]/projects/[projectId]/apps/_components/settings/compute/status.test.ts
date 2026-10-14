import { describe, expect, it } from "vitest";
import { regionChoices } from "./status";

const regions = [
  { name: "us-east-1", canSchedule: true },
  { name: "eu-central-1", canSchedule: false },
  { name: "us-west-2", canSchedule: true },
];
const ready = { status: "ready" as const, regions };

describe("regionChoices", () => {
  it("locks the only selected region", () => {
    expect(regionChoices(ready, ["us-west-2"]).at(-1)).toEqual({
      name: "us-west-2",
      state: "last",
    });
  });

  it("keeps selected regions that the available list does not include", () => {
    expect(regionChoices(ready, ["us-east-1", "local"]).at(-1)).toEqual({
      name: "local",
      state: "on",
    });
    expect(regionChoices({ status: "loading" }, ["us-east-1"])).toEqual([
      { name: "us-east-1", state: "last" },
    ]);
  });
});
