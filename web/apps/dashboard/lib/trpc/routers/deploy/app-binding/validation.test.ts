import { describe, expect, it } from "vitest";
import {
  bindingEndpointsSchema,
  bindingHost,
  bindingHostVariable,
  bindingNameSchema,
  defaultBindingName,
} from "./validation";

describe("app binding validation", () => {
  it("only connects different apps", () => {
    const endpoints = { appId: "caller", environmentId: "prod", targetAppId: "caller" };
    expect(bindingEndpointsSchema.safeParse(endpoints).success).toBe(false);
    expect(bindingEndpointsSchema.safeParse({ ...endpoints, targetAppId: "api" }).success).toBe(
      true,
    );
  });

  it("accepts DNS labels that start with a letter", () => {
    expect(bindingNameSchema.safeParse("payments-api").success).toBe(true);
    expect(bindingNameSchema.safeParse("db2").success).toBe(true);
    for (const invalid of ["Payments", "-payments", "payments-", "2db", "with_underscore", ""]) {
      expect(bindingNameSchema.safeParse(invalid).success).toBe(false);
    }
    expect(bindingNameSchema.safeParse("x".repeat(64)).success).toBe(false);
  });

  it("reserves names that would inject UNKEY_ variables", () => {
    for (const reserved of ["unkey", "unkey-api", "unkeyed"]) {
      expect(bindingNameSchema.safeParse(reserved).success).toBe(false);
    }
  });

  it("derives a hostname and a host variable without a port or scheme", () => {
    expect(bindingHost("event-store")).toBe("event-store.unkey.internal");
    expect(bindingHostVariable("event-store")).toBe("EVENT_STORE_HOST");
    expect(bindingHost("event-store")).not.toContain(":");
  });
});

describe("defaultBindingName", () => {
  it("uses the target slug when it is free", () => {
    expect(defaultBindingName("responder", () => false)).toBe("responder");
  });

  it("adds a suffix when the name or its variable is taken", () => {
    const taken = new Set(["responder", "responder-2"]);
    expect(defaultBindingName("responder", (name) => taken.has(name))).toBe("responder-3");
  });

  it("repairs slugs that are not valid binding names", () => {
    expect(defaultBindingName("1password", () => false)).toBe("app-1password");
    expect(defaultBindingName("unkey-api", () => false)).toBe("app-unkey-api");
    expect(defaultBindingName("My_Service", () => false)).toBe("my-service");
    const long = defaultBindingName("a".repeat(80), () => false);
    expect(long).toHaveLength(63);
  });

  it("gives up instead of looping when every candidate is taken", () => {
    expect(defaultBindingName("responder", () => true)).toBeUndefined();
  });
});
