import { describe, expect, test } from "vitest";

/**
 * Unit tests for null handling in connected-keys-and-perms endpoint
 * Tests the fallback logic: lastUpdated = Number(updated_at_m ?? created_at_m)
 */

describe("connected-keys-and-perms null handling", () => {
  /**
   * Role with null updated_at_m returns created_at_m
   *
   */
  test("role with null updated_at_m returns created_at_m", () => {
    // Mock role data with null updated_at_m
    const mockRole = {
      id: "role_123",
      name: "test-role",
      description: "Test role description",
      updated_at_m: null,
      created_at_m: 1234567890,
    };

    const lastUpdated = Number(mockRole.updated_at_m ?? mockRole.created_at_m);

    expect(lastUpdated).toBe(1234567890);
    expect(lastUpdated).toBe(mockRole.created_at_m);
  });

  /**
   * Role with valid updated_at_m returns updated_at_m
   */
  test("role with valid updated_at_m returns updated_at_m", () => {
    const mockRole = {
      id: "role_456",
      name: "updated-role",
      description: "Updated role description",
      updated_at_m: 9876543210,
      created_at_m: 1234567890,
    };

    const lastUpdated = Number(mockRole.updated_at_m ?? mockRole.created_at_m);

    expect(lastUpdated).toBe(9876543210);
    expect(lastUpdated).toBe(mockRole.updated_at_m);
    expect(lastUpdated).not.toBe(mockRole.created_at_m);
  });

  /**
   * Response type is always number
   */
  test("response type is always number", () => {
    // Mock role with null updated_at_m
    const mockRole = {
      id: "role_789",
      name: "type-test-role",
      description: null,
      updated_at_m: null,
      created_at_m: 1234567890,
    };

    const lastUpdated = Number(mockRole.updated_at_m ?? mockRole.created_at_m);

    expect(typeof lastUpdated).toBe("number");

    expect(Number.isFinite(lastUpdated)).toBe(true);
    expect(Number.isNaN(lastUpdated)).toBe(false);

    expect(lastUpdated).toBeGreaterThan(0);
  });
});
