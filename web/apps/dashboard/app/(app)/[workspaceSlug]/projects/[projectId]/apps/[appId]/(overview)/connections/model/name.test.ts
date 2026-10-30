import { newId } from "@unkey/id";
import { describe, expect, it } from "vitest";
import {
  connectionHostVariable,
  connectionNameSchema,
  defaultConnectionName,
  takenNamesOf,
} from "./name";
import type { ListedConnection } from "./target";

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

describe("defaultConnectionName", () => {
  const free = () => false;

  it.each([
    ["checkout", "checkout"],
    ["Event Store", "event-store"],
    ["my__api--v2", "my-api-v2"],
    ["2fa", "app-2fa"],
    ["unkey-proxy", "app-unkey-proxy"],
    ["-edge-", "edge"],
  ])("derives a valid name from the slug %j", (slug, name) => {
    expect(defaultConnectionName(slug, free)).toBe(name);
  });

  it("suffixes a number when the name is taken", () => {
    const taken = new Set(["checkout", "checkout-2"]);
    expect(defaultConnectionName("checkout", (name) => taken.has(name))).toBe("checkout-3");
  });

  it("keeps a suffixed name within 63 characters", () => {
    const slug = "a".repeat(70);
    expect(defaultConnectionName(slug, (name) => name === "a".repeat(63))).toBe(
      `${"a".repeat(61)}-2`,
    );
  });

  it("gives up after 20 attempts", () => {
    expect(defaultConnectionName("checkout", () => true)).toBeUndefined();
  });
});

describe("takenNamesOf", () => {
  const appId = newId("app");
  const db = newId("app");
  const api = newId("app");
  const row = (targetAppId: string, name: string) =>
    ({
      id: newId("connection"),
      appId,
      environmentId: newId("environment"),
      targetAppId,
      targetAppName: name,
      targetAppSlug: name,
      name,
      targetType: "automatic",
    }) satisfies ListedConnection;

  it("collects the names of the other connections, not the one being saved", () => {
    expect(takenNamesOf([row(db, "db"), row(api, "api")], db)).toEqual(new Set(["api"]));
  });
});
