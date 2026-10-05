import { describe, expect, it } from "vitest";
import { conflictErrors, parseVariableRows } from "./variable-rows";

describe("parseVariableRows", () => {
  it("skips blank rows and trims values", () => {
    expect(
      parseVariableRows([
        { key: "DATABASE_URL", value: " postgres://db \n", sensitive: false },
        { key: "API_TOKEN", value: "secret", sensitive: true },
        { key: "", value: "", sensitive: false },
      ]),
    ).toEqual({
      ok: true,
      variables: [
        { key: "DATABASE_URL", value: "postgres://db", kind: "recoverable" },
        { key: "API_TOKEN", value: "secret", kind: "writeonly" },
      ],
    });
  });

  it("accepts no variables at all", () => {
    expect(parseVariableRows([{ key: " ", value: "", sensitive: false }])).toEqual({
      ok: true,
      variables: [],
    });
  });

  it("reports each invalid field by row", () => {
    const parsed = parseVariableRows([
      { key: "1BAD", value: "x", sensitive: false },
      { key: "TOKEN", value: "", sensitive: false },
    ]);
    expect(parsed.ok).toBe(false);
    expect(parsed.ok ? null : Object.fromEntries(parsed.errors)).toEqual({
      0: {
        key: "Only letters, digits, and underscores are allowed, and the name must not start with a digit",
        value: undefined,
      },
      1: { key: undefined, value: "Variable value is required" },
    });
  });

  it("rejects a repeated name", () => {
    const parsed = parseVariableRows([
      { key: "PORT", value: "1", sensitive: false },
      { key: "PORT", value: "2", sensitive: false },
    ]);
    expect(parsed.ok ? null : Object.fromEntries(parsed.errors)).toEqual({
      1: { key: "This name is already in the list. Use a different name.", value: undefined },
    });
  });
});

describe("conflictErrors", () => {
  it("flags rows whose name is already set", () => {
    const rows = [
      { key: "PORT", value: "1", sensitive: false },
      { key: " TOKEN ", value: "2", sensitive: false },
    ];
    expect(Object.fromEntries(conflictErrors(rows, new Set(["TOKEN"])))).toEqual({
      1: { key: "A variable with this name already exists. Use a different name." },
    });
  });
});
