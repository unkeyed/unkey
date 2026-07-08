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

  it("still requires a non-empty value and rejects literal newline escapes", () => {
    expect(envVarValueSchema.safeParse("ok").success).toBe(true);
    expect(envVarValueSchema.safeParse("").success).toBe(false);
    expect(envVarValueSchema.safeParse("line\\nbreak").success).toBe(false);
  });
});
