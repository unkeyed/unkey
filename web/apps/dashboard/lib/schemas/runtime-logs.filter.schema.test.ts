import { describe, expect, it } from "vitest";
import {
  parseRuntimeLogsAttributeMatch,
  runtimeLogsFilterOutputSchema,
} from "./runtime-logs.filter.schema";

describe("parseRuntimeLogsAttributeMatch", () => {
  it("parses a dotted path and preserves equals signs in the value", () => {
    expect(parseRuntimeLogsAttributeMatch(" request . id = token=xyz ")).toEqual({
      path: "request.id",
      value: "token=xyz",
    });
  });

  it.each(["request.id", "= xyz", "request..id = xyz", "request.id = xy"])(
    "rejects invalid match %s",
    (input) => {
      expect(parseRuntimeLogsAttributeMatch(input)).toBeNull();
    },
  );
});

describe("runtimeLogsFilterOutputSchema", () => {
  it("accepts exact attribute matches from LLM search", () => {
    expect(
      runtimeLogsFilterOutputSchema.safeParse({
        filters: [
          {
            field: "attributes",
            filters: [{ operator: "is", value: "request.id = xyz" }],
          },
        ],
      }).success,
    ).toBe(true);
  });
});
