import { describe, expect, it } from "vitest";
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
  it("bounds the value by UTF-8 bytes, not characters, so the encrypted ciphertext cannot overflow the TEXT column", () => {
    // A 16384-byte ASCII value is the boundary and must pass.
    expect(envVarValueSchema.safeParse("a".repeat(16384)).success).toBe(true);
    expect(envVarValueSchema.safeParse("a".repeat(16385)).success).toBe(false);

    // The guarantee that distinguishes this from .max() on string length:
    // 5462 three-byte characters is only 5462 chars but 16386 bytes, over the
    // cap. A character-based cap would wrongly accept it.
    const multibyte = "あ".repeat(5462);
    expect(multibyte.length).toBe(5462);
    expect(new TextEncoder().encode(multibyte).length).toBe(16386);
    expect(envVarValueSchema.safeParse(multibyte).success).toBe(false);
  });

  it("allows multi-line LF values but rejects carriage returns", () => {
    expect(envVarValueSchema.safeParse("ok").success).toBe(true);
    expect(envVarValueSchema.safeParse("").success).toBe(false);

    // Real LF newlines are allowed so multi-line PEM keys, certs, and JSON pass.
    expect(envVarValueSchema.safeParse("line1\nline2").success).toBe(true);
    const pem = "-----BEGIN PRIVATE KEY-----\nMIIabc/def\n-----END PRIVATE KEY-----";
    expect(envVarValueSchema.safeParse(pem).success).toBe(true);

    // CRLF is rejected so stored values stay canonical LF.
    expect(envVarValueSchema.safeParse("a\r\nb").success).toBe(false);

    // A literal backslash-n is plain text, not a newline, so it now passes;
    // the old refine wrongly rejected it while letting real newlines through.
    expect(envVarValueSchema.safeParse("line\\nbreak").success).toBe(true);
  });
});
