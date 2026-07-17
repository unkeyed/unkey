import { describe, expect, it } from "vitest";
import { identityExternalIdSchema } from "./identity";
import { identityMetadataSchema, metadataSchema, parseIdentityMetadata } from "./metadata";
import { ratelimitSchema } from "./ratelimit";

const rateLimit = {
  name: "api",
  refillInterval: 1_000,
  limit: 10,
  autoApply: true,
};

describe("identity form schemas", () => {
  it("accepts external IDs supported by the identities API", () => {
    expect(identityExternalIdSchema.parse(" user_123.example-test ")).toBe("user_123.example-test");
  });

  it("rejects external IDs unsupported by the identities API", () => {
    expect(identityExternalIdSchema.safeParse("user@example.com").success).toBe(false);
    expect(identityExternalIdSchema.safeParse(" ").success).toBe(false);
    expect(identityExternalIdSchema.safeParse("a".repeat(256)).success).toBe(false);
    expect(identityExternalIdSchema.safeParse("a".repeat(255)).success).toBe(true);
  });

  it("accepts metadata objects with up to 100 properties", () => {
    const metadata = Object.fromEntries(
      Array.from({ length: 100 }, (_, index) => [`key-${index}`, index]),
    );

    expect(
      identityMetadataSchema.safeParse({
        metadata: { enabled: true, data: JSON.stringify(metadata) },
      }).success,
    ).toBe(true);
    expect(parseIdentityMetadata(JSON.stringify(metadata))).toEqual(metadata);
  });

  it("rejects non-object metadata and objects over the API limit", () => {
    const metadata = Object.fromEntries(
      Array.from({ length: 101 }, (_, index) => [`key-${index}`, index]),
    );

    expect(
      identityMetadataSchema.safeParse({ metadata: { enabled: true, data: "[]" } }).success,
    ).toBe(false);
    expect(
      identityMetadataSchema.safeParse({ metadata: { enabled: true, data: "null" } }).success,
    ).toBe(false);
    expect(
      identityMetadataSchema.safeParse({
        metadata: { enabled: true, data: JSON.stringify(metadata) },
      }).success,
    ).toBe(false);
    expect(() => parseIdentityMetadata("[]")).toThrow();
    expect(() => parseIdentityMetadata("null")).toThrow();
    expect(() => parseIdentityMetadata(JSON.stringify(metadata))).toThrow();
  });

  it("rejects malformed metadata and serialized values over 1 MiB", () => {
    expect(
      identityMetadataSchema.safeParse({ metadata: { enabled: true, data: "{" } }).success,
    ).toBe(false);
    expect(
      identityMetadataSchema.safeParse({
        metadata: { enabled: true, data: JSON.stringify({ value: "é".repeat(524_288) }) },
      }).success,
    ).toBe(false);
  });

  it("keeps key metadata within MySQL TEXT storage limits", () => {
    expect(
      metadataSchema.safeParse({
        metadata: { enabled: true, data: JSON.stringify({ value: "a".repeat(65_000) }) },
      }).success,
    ).toBe(true);
    expect(
      metadataSchema.safeParse({
        metadata: { enabled: true, data: JSON.stringify({ value: "a".repeat(66_000) }) },
      }).success,
    ).toBe(false);
  });

  it("enforces the API rate-limit name and item bounds", () => {
    expect(
      ratelimitSchema.safeParse({
        ratelimit: {
          enabled: true,
          data: Array.from({ length: 50 }, (_, index) => ({
            ...rateLimit,
            name: `rule-${index}`,
          })),
        },
      }).success,
    ).toBe(true);
    expect(
      ratelimitSchema.safeParse({
        ratelimit: {
          enabled: true,
          data: Array.from({ length: 51 }, (_, index) => ({
            ...rateLimit,
            name: `rule-${index}`,
          })),
        },
      }).success,
    ).toBe(false);
    expect(
      ratelimitSchema.safeParse({
        ratelimit: { enabled: true, data: [{ ...rateLimit, name: "a".repeat(128) }] },
      }).success,
    ).toBe(true);
    expect(
      ratelimitSchema.safeParse({
        ratelimit: { enabled: true, data: [{ ...rateLimit, name: "a".repeat(129) }] },
      }).success,
    ).toBe(false);
    expect(
      ratelimitSchema.safeParse({
        ratelimit: { enabled: true, data: [{ ...rateLimit, limit: 1.5 }] },
      }).success,
    ).toBe(false);
    expect(
      ratelimitSchema.safeParse({
        ratelimit: { enabled: true, data: [{ ...rateLimit, refillInterval: 1_000.5 }] },
      }).success,
    ).toBe(false);
  });

  it("reports duplicate rate-limit names on the duplicate field", () => {
    const result = ratelimitSchema.safeParse({
      ratelimit: {
        enabled: true,
        data: [rateLimit, rateLimit],
      },
    });

    expect(result.success).toBe(false);
    if (result.success) {
      throw new Error("Duplicate rate-limit names must fail validation");
    }
    expect(result.error.issues).toContainEqual(
      expect.objectContaining({ path: ["ratelimit", "data", 1, "name"] }),
    );
  });
});
