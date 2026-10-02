import { describe, expect, test } from "vitest";
import { shortenId } from "./shorten-id";

describe("shortenId", () => {
  test("keeps the prefix and trims the identifier", () => {
    expect(shortenId("key_1234567890abcdef")).toBe("key_1234...cdef");
  });

  test("returns short ids untouched", () => {
    expect(shortenId("d_pro_api_08")).toBe("d_pro_api_08");
  });

  test("splits at the first underscore only", () => {
    expect(shortenId("d_pro_api_1234567890abcdef")).toBe("d_pro_...cdef");
  });

  test("handles ids without a prefix", () => {
    expect(shortenId("1234567890abcdef")).toBe("1234...cdef");
  });
});
