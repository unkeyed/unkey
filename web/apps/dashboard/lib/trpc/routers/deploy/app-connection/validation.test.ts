import { describe, expect, it } from "vitest";
import {
  connectionEndpointsSchema,
  connectionHost,
  connectionHostVariable,
  connectionNameSchema,
  defaultConnectionName,
} from "./validation";

describe("app connection validation", () => {
  it("only connects different apps", () => {
    const endpoints = { appId: "caller", environmentId: "prod", targetAppId: "caller" };
    expect(connectionEndpointsSchema.safeParse(endpoints).success).toBe(false);
    expect(connectionEndpointsSchema.safeParse({ ...endpoints, targetAppId: "api" }).success).toBe(
      true,
    );
  });

  it("accepts DNS labels that start with a letter", () => {
    expect(connectionNameSchema.safeParse("payments-api").success).toBe(true);
    expect(connectionNameSchema.safeParse("db2").success).toBe(true);
    for (const invalid of ["Payments", "-payments", "payments-", "2db", "with_underscore", ""]) {
      expect(connectionNameSchema.safeParse(invalid).success).toBe(false);
    }
    expect(connectionNameSchema.safeParse("x".repeat(64)).success).toBe(false);
  });

  it("reserves names that would inject UNKEY_ variables", () => {
    for (const reserved of ["unkey", "unkey-api", "unkeyed"]) {
      expect(connectionNameSchema.safeParse(reserved).success).toBe(false);
    }
  });

  it("derives a hostname and a host variable without a port or scheme", () => {
    expect(connectionHost("event-store")).toBe("event-store.unkey.internal");
    expect(connectionHostVariable("event-store")).toBe("EVENT_STORE_HOST");
    expect(connectionHost("event-store")).not.toContain(":");
  });
});

describe("defaultConnectionName", () => {
  it("uses the target slug when it is free", () => {
    expect(defaultConnectionName("responder", () => false)).toBe("responder");
  });

  it("adds a suffix when the connection name is taken", () => {
    const taken = new Set(["responder", "responder-2"]);
    expect(defaultConnectionName("responder", (name) => taken.has(name))).toBe("responder-3");
  });

  it("repairs slugs that are not valid connection names", () => {
    expect(defaultConnectionName("1password", () => false)).toBe("app-1password");
    expect(defaultConnectionName("unkey-api", () => false)).toBe("app-unkey-api");
    expect(defaultConnectionName("My_Service", () => false)).toBe("my-service");
    const long = defaultConnectionName("a".repeat(80), () => false);
    expect(long).toHaveLength(63);
  });

  it("gives up instead of looping when every candidate is taken", () => {
    expect(defaultConnectionName("responder", () => true)).toBeUndefined();
  });
});
