import { describe, expect, it } from "vitest";
import { connectionHostVariable, connectionNameSchema } from "./connection-rules";

describe("app connection validation", () => {
  it.each([
    "responder",
    "event-store",
    "db2",
    "a",
    "a--b",
    "unke",
    "self-2",
    "selfish",
    "a".repeat(63),
  ])("accepts the DNS label %s", (name) => {
    expect(connectionNameSchema.safeParse(name).success).toBe(true);
  });

  it.each([
    "",
    "Payments",
    "-payments",
    "payments-",
    "2db",
    "with_underscore",
    "api.service",
    "api service",
    "self",
    " self ",
    "unkey",
    "unkey-api",
    "unkeyed",
    "a".repeat(64),
  ])("rejects %j", (name) => {
    expect(connectionNameSchema.safeParse(name).success).toBe(false);
  });

  it.each([
    ["responder", "RESPONDER_HOST"],
    ["event-store", "EVENT_STORE_HOST"],
    ["db2", "DB2_HOST"],
    ["a--b", "A__B_HOST"],
  ])("derives the host variable for %s", (name, key) => {
    expect(connectionHostVariable(name)).toBe(key);
  });
});
