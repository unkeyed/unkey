import { describe, expect, it } from "vitest";
import { parseEnvText } from "./use-drop-zone";

describe("parseEnvText", () => {
  it("parses simple key=value pairs and skips blanks and comments", () => {
    const { entries } = parseEnvText(
      ["# a comment", "", "FOO=bar", "  BAZ = qux  ", "# trailing comment"].join("\n"),
    );
    expect(entries).toEqual([
      { key: "FOO", value: "bar" },
      { key: "BAZ", value: "qux" },
    ]);
  });

  it("skips lines without an equals sign instead of failing the whole import", () => {
    const { entries } = parseEnvText(["FOO=bar", "not a pair", "BAZ=qux"].join("\n"));
    expect(entries).toEqual([
      { key: "FOO", value: "bar" },
      { key: "BAZ", value: "qux" },
    ]);
  });

  it("strips matching outer quotes from single-line values", () => {
    const { entries } = parseEnvText(
      ['DOUBLE="hello world"', "SINGLE='a b c'", "BARE=plain"].join("\n"),
    );
    expect(entries).toEqual([
      { key: "DOUBLE", value: "hello world" },
      { key: "SINGLE", value: "a b c" },
      { key: "BARE", value: "plain" },
    ]);
  });

  it("accumulates a multi-line double-quoted value until the closing quote", () => {
    const { entries } = parseEnvText(
      ['TLS_KEY="-----BEGIN KEY-----', "line-two", '-----END KEY-----"', "NEXT=ok"].join("\n"),
    );
    expect(entries).toEqual([
      { key: "TLS_KEY", value: "-----BEGIN KEY-----\nline-two\n-----END KEY-----" },
      { key: "NEXT", value: "ok" },
    ]);
  });

  it("accumulates a multi-line single-quoted PEM value", () => {
    const pem = [
      "PEM='-----BEGIN PRIVATE KEY-----",
      "MIIabc/def+123==",
      "-----END PRIVATE KEY-----'",
    ].join("\n");
    const { entries } = parseEnvText(pem);
    expect(entries).toEqual([
      {
        key: "PEM",
        value: "-----BEGIN PRIVATE KEY-----\nMIIabc/def+123==\n-----END PRIVATE KEY-----",
      },
    ]);
  });

  it("takes the rest of the input when a quoted value is never closed", () => {
    const { entries } = parseEnvText(["KEY='unterminated", "second line"].join("\n"));
    expect(entries).toEqual([{ key: "KEY", value: "unterminated\nsecond line" }]);
  });

  it("normalizes CRLF to LF so multi-line values store canonical newlines", () => {
    const { entries } = parseEnvText('PEM="line1\r\nline2"\r\nNEXT=ok\r\n');
    expect(entries).toEqual([
      { key: "PEM", value: "line1\nline2" },
      { key: "NEXT", value: "ok" },
    ]);
  });
});
