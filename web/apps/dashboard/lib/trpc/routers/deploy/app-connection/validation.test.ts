import { newId } from "@unkey/id";
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
    const appId = newId("app");
    const endpoints = { appId, environmentId: newId("environment"), targetAppId: appId };
    expect(connectionEndpointsSchema.safeParse(endpoints).success).toBe(false);
    expect(
      connectionEndpointsSchema.safeParse({ ...endpoints, targetAppId: newId("app") }).success,
    ).toBe(true);
  });

  it.each(["responder", "event-store", "db2", "a", "a--b", "unke", "a".repeat(63)])(
    "accepts the DNS label %s",
    (name) => {
      expect(connectionNameSchema.safeParse(name).success).toBe(true);
    },
  );

  it.each([
    "",
    "Payments",
    "-payments",
    "payments-",
    "2db",
    "with_underscore",
    "api.service",
    "api service",
    "unkey",
    "unkey-api",
    "unkeyed",
    "a".repeat(64),
  ])("rejects %j", (name) => {
    expect(connectionNameSchema.safeParse(name).success).toBe(false);
  });

  it.each([
    ["responder", "RESPONDER_HOST", "responder.unkey.internal"],
    ["db2", "DB2_HOST", "db2.unkey.internal"],
    ["a--b", "A__B_HOST", "a--b.unkey.internal"],
  ])("derives the host variable for %s", (name, key, host) => {
    expect(connectionHostVariable(name)).toBe(key);
    expect(connectionHost(name)).toBe(host);
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
