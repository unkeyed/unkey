import { describe, expect, it } from "vitest";
import {
  type VariableRow,
  conflictErrors,
  importVariableEntries,
  parseVariableRows,
  pasteVariableEntriesAt,
} from "./variable-rows";

function row(id: string, key: string, value: string, sensitive = false): VariableRow {
  return { id, key, value, sensitive };
}

const fresh = (key: string, value: string) => ({
  id: expect.any(String),
  key,
  value,
  sensitive: false,
});

describe("parseVariableRows", () => {
  it("skips blank rows and trims values", () => {
    expect(
      parseVariableRows([
        row("r1", "DATABASE_URL", " postgres://db \n"),
        row("r2", "API_TOKEN", "secret", true),
        row("r3", "", ""),
      ]),
    ).toEqual({
      ok: true,
      variables: [
        { key: "DATABASE_URL", value: "postgres://db", kind: "recoverable" },
        { key: "API_TOKEN", value: "secret", kind: "writeonly" },
      ],
    });
  });

  it("reports each invalid field by row id", () => {
    const parsed = parseVariableRows([row("r1", "1BAD", "x"), row("r2", "TOKEN", "")]);
    expect(parsed.ok).toBe(false);
    expect(parsed.ok ? null : Object.fromEntries(parsed.errors)).toEqual({
      r1: {
        key: "Only letters, digits, and underscores are allowed, and the name must not start with a digit",
        value: undefined,
      },
      r2: { key: undefined, value: "Variable value is required" },
    });
  });

  it("rejects a repeated name", () => {
    const parsed = parseVariableRows([row("r1", "PORT", "1"), row("r2", "PORT", "2")]);
    expect(parsed.ok ? null : Object.fromEntries(parsed.errors)).toEqual({
      r2: {
        key: "This key appears more than once. Rename or remove the extra rows.",
        value: undefined,
      },
    });
  });
});

describe("conflictErrors", () => {
  it("flags rows whose name is already set", () => {
    const rows = [row("r1", "PORT", "1"), row("r2", " TOKEN ", "2")];
    expect(Object.fromEntries(conflictErrors(rows, new Set(["TOKEN"])))).toEqual({
      r2: { key: "A variable with this key already exists." },
    });
  });
});

describe("importVariableEntries", () => {
  it("replaces blank rows and adds imported variables as not sensitive", () => {
    expect(
      importVariableEntries(
        [row("r1", "TOKEN", "a", true), row("r2", "", "", true)],
        [
          { key: "PORT", value: "3000" },
          { key: "TOKEN", value: "b" },
          { key: "PORT", value: "8080" },
        ],
      ),
    ).toEqual({
      rows: [row("r1", "TOKEN", "a", true), fresh("PORT", "8080")],
      added: 1,
      skipped: 1,
    });
  });

  it("keeps a row that has a value but no name", () => {
    expect(
      importVariableEntries([row("r1", "", "draft")], [{ key: "PORT", value: "3000" }]),
    ).toEqual({
      rows: [row("r1", "", "draft"), fresh("PORT", "3000")],
      added: 1,
      skipped: 0,
    });
  });

  it("matches a typed key with surrounding spaces, as saving does", () => {
    const rows = [row("r1", " PORT ", "3000")];
    expect(importVariableEntries(rows, [{ key: "PORT", value: "8080" }])).toEqual({
      rows,
      added: 0,
      skipped: 1,
    });
  });
});

describe("pasteVariableEntriesAt", () => {
  it("fills the pasted row and appends the rest", () => {
    expect(
      pasteVariableEntriesAt([row("r1", "", "", true), row("r2", "LAST", "z")], "r1", [
        { key: "A", value: "1" },
        { key: "B", value: "2" },
      ]),
    ).toEqual([row("r1", "A", "1", true), row("r2", "LAST", "z"), fresh("B", "2")]);
  });
});
