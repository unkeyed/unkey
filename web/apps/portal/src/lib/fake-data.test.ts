import { describe, expect, test } from "vitest";
import { fakeKeyUsage, fakeVerifications } from "./fake-data";

describe("fakeVerifications", () => {
  test("matches the API's inclusive bucket count", () => {
    expect(fakeVerifications(1, "populated")).toHaveLength(25);
    expect(fakeVerifications(7, "populated")).toHaveLength(8);
    expect(fakeVerifications(30, "empty")).toHaveLength(31);
  });

  test("valid and error sum to total", () => {
    for (const bucket of fakeVerifications(7, "populated")) {
      expect(bucket.valid + bucket.error).toBe(bucket.total);
      expect(bucket.error).toBeLessThanOrEqual(bucket.total);
    }
  });

  test("is deterministic", () => {
    expect(fakeVerifications(7, "populated")).toEqual(fakeVerifications(7, "populated"));
  });
});

describe("fakeKeyUsage", () => {
  test("returns 30 daily buckets with errors bounded by usage", () => {
    const { usage, errors } = fakeKeyUsage("key_abc", "populated");
    expect(usage).toHaveLength(30);
    expect(errors).toHaveLength(30);
    errors.forEach((err, i) => expect(err).toBeLessThanOrEqual(usage[i]));
  });

  test("differs per key and is stable per key", () => {
    expect(fakeKeyUsage("key_a", "populated")).toEqual(fakeKeyUsage("key_a", "populated"));
    expect(fakeKeyUsage("key_a", "populated").usage).not.toEqual(
      fakeKeyUsage("key_b", "populated").usage,
    );
  });
});
