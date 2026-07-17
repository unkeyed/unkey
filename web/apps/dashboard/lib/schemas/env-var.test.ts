import { describe, expect, it } from "vitest";
import { z } from "zod";
import { envVarKeySchema, envVarValueSchema } from "./env-var";

describe("envVarKeySchema", () => {
  it("rejects keys longer than the varchar(256) column", () => {
    expect(envVarKeySchema.safeParse("A".repeat(256)).success).toBe(true);
    expect(envVarKeySchema.safeParse("A".repeat(257)).success).toBe(false);
  });

  it("still enforces the POSIX name rule and non-empty", () => {
    expect(envVarKeySchema.safeParse("VALID_NAME_1").success).toBe(true);
    expect(envVarKeySchema.safeParse("1leading_digit").success).toBe(false);
    expect(envVarKeySchema.safeParse("has-dash").success).toBe(false);
    expect(envVarKeySchema.safeParse("").success).toBe(false);
  });
});

describe("envVarValueSchema", () => {
  it("bounds the value by UTF-8 bytes, not characters, so the encrypted ciphertext cannot overflow the varchar(4096) column", () => {
    // A 3000-byte ASCII value is the boundary and must pass.
    expect(envVarValueSchema.safeParse("a".repeat(3000)).success).toBe(true);
    expect(envVarValueSchema.safeParse("a".repeat(3001)).success).toBe(false);

    // The guarantee that distinguishes this from .max() on string length:
    // 1500 three-byte characters is only 1500 chars but 4500 bytes, which would
    // overflow the column. A character-based cap would wrongly accept it.
    const multibyte = "あ".repeat(1500);
    expect(multibyte.length).toBe(1500);
    expect(new TextEncoder().encode(multibyte).length).toBe(4500);
    expect(envVarValueSchema.safeParse(multibyte).success).toBe(false);
  });

  it("still requires a non-empty value", () => {
    expect(envVarValueSchema.safeParse("ok").success).toBe(true);
    expect(envVarValueSchema.safeParse("").success).toBe(false);
  });

  it("rejects embedded newlines, which would corrupt the .env file the build shell-sources", () => {
    expect(envVarValueSchema.safeParse("line\nbreak").success).toBe(false);
    expect(envVarValueSchema.safeParse("line\rbreak").success).toBe(false);
    expect(envVarValueSchema.safeParse("line\r\nbreak").success).toBe(false);

    // A multi-line PEM key is the realistic case: it passed validation before,
    // then failed the build as a non-retryable terminal error.
    const pem = "-----BEGIN PRIVATE KEY-----\nMIIEvQIBADAN\n-----END PRIVATE KEY-----";
    expect(envVarValueSchema.safeParse(pem).success).toBe(false);
  });

  it("tells the user to replace the value, since an <input> hides the newlines being complained about", () => {
    // Values with real newlines predate this check, so the edit row can load one
    // and reject it. The input strips CR/LF from display, so a message stating
    // only the rule would point at a field that looks clean.
    const parsed = envVarValueSchema.safeParse("line\nbreak");
    expect(parsed.success).toBe(false);
    expect(parsed.success === false && parsed.error.issues[0].message).toBe(
      "Newline characters are not allowed. Replace this with a single-line value.",
    );
  });

  it("keeps that message intact through the .or(literal('')) union the edit row composes", () => {
    // The edit row (env-var-edit-row.tsx) wraps this schema in a union to allow
    // an untouched empty field. A union whose branches all fail can report a
    // generic "Invalid input" instead of the branch's own message, which would
    // strand the replacement instruction in the one UI that most needs it.
    const editRowValueSchema = envVarValueSchema.or(z.literal(""));

    expect(editRowValueSchema.safeParse("").success).toBe(true);

    const parsed = editRowValueSchema.safeParse("line\nbreak");
    expect(parsed.success).toBe(false);
    expect(parsed.success === false && parsed.error.issues[0].message).toBe(
      "Newline characters are not allowed. Replace this with a single-line value.",
    );
  });

  it("accepts the two-character sequence \\n, which is not a newline", () => {
    // These were wrongly rejected with "Newline characters are not allowed"
    // because the check tested for a literal backslash followed by n.
    expect(envVarValueSchema.safeParse("C:\\new\\folder").success).toBe(true);
    expect(envVarValueSchema.safeParse("split on \\n").success).toBe(true);
    expect(envVarValueSchema.safeParse('{"sep":"\\r\\n"}').success).toBe(true);
  });

  it("accepts values whose only newlines are leading or trailing, since trim removes them", () => {
    const parsed = envVarValueSchema.safeParse("\n  token  \n");
    expect(parsed.success).toBe(true);
    expect(parsed.success && parsed.data).toBe("token");
  });
});
