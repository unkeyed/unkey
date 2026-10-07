import { describe, expect, it } from "vitest";
import { FIREWALL_SUMMARY } from "./firewall/model";
import { keyauth, keyauthSummary } from "./keyauth/model";
import { logging, loggingSummary } from "./logging/model";
import { OPENAPI_SUMMARY } from "./openapi/model";
import { ratelimit, ratelimitSummary } from "./ratelimit/model";

const names = { ks_checkout: "Checkout keys", ks_partner: "Partner keys" };

describe("policy summaries", () => {
  it("keyauth with an empty config", () => {
    expect(keyauthSummary(keyauth.defaults(), names)).toBe("No keyspace · Bearer token");
  });

  it("keyauth with keyspaces, a header location and credits", () => {
    const values = {
      ...keyauth.defaults(),
      type: "keyauth" as const,
      keyspaceIds: ["ks_checkout", "ks_unknown"],
      locations: [{ id: "l1", locationType: "header" as const, name: "Authorization" }],
      permissionQuery: "",
      ratelimits: [],
      credits: 1,
    };
    expect(keyauthSummary(values, names)).toBe(
      "Checkout keys, ks_unknown · header Authorization · 1 credit",
    );
  });

  it("keyauth with a nameless query param location and zero credits", () => {
    const values = {
      ...keyauth.defaults(),
      type: "keyauth" as const,
      keyspaceIds: ["ks_partner"],
      locations: [{ id: "l1", locationType: "queryParam" as const, name: "" }],
      permissionQuery: "",
      ratelimits: [],
      credits: 0,
    };
    expect(keyauthSummary(values, names)).toBe("Partner keys · query ?… · 0 credits");
  });

  it("keyauth with several locations, rate limits and a permission query", () => {
    const values = {
      ...keyauth.defaults(),
      type: "keyauth" as const,
      keyspaceIds: ["ks_checkout"],
      locations: [
        { id: "l1", locationType: "bearer" as const },
        { id: "l2", locationType: "header" as const, name: "X-API-Key" },
        { id: "l3", locationType: "queryParam" as const, name: "api_key" },
      ],
      permissionQuery: "api.read",
      ratelimits: [
        { id: "r1", name: "requests", override: false },
        { id: "r2", name: "tokens", override: false },
      ],
      credits: undefined,
    };
    expect(keyauthSummary(values, names)).toBe(
      "Checkout keys · Bearer token +2 more · 2 rate limits · permissions",
    );
  });

  it("keyauth with one rate limit and a blank permission query", () => {
    const values = {
      ...keyauth.defaults(),
      type: "keyauth" as const,
      keyspaceIds: ["ks_checkout"],
      locations: [{ id: "l1", locationType: "header" as const, name: "X-API-Key" }],
      permissionQuery: "   ",
      ratelimits: [{ id: "r1", name: "requests", override: false }],
      credits: 2,
    };
    expect(keyauthSummary(values, names)).toBe(
      "Checkout keys · header X-API-Key · 2 credits · 1 rate limit",
    );
  });

  it("ratelimit with a limit of one request", () => {
    const values = {
      ...ratelimit.defaults(),
      type: "ratelimit" as const,
      limit: 1,
      windowMs: 1000,
      identifiers: [{ id: "a", source: "path" as const, value: "" }],
    };
    expect(ratelimitSummary(values)).toBe("1 request per 1s · per path");
  });

  it("ratelimit with the default config", () => {
    expect(ratelimitSummary(ratelimit.defaults())).toBe("100 requests per 1m · per client IP");
  });

  it("ratelimit with a compound identifier", () => {
    const values = {
      type: "ratelimit" as const,
      name: "",
      environmentId: "__all__",
      matchConditions: [],
      limit: 5,
      windowMs: 1000,
      identifiers: [
        { id: "a", source: "authenticatedSubject" as const, value: "" },
        { id: "b", source: "header" as const, value: "X-Tenant-Id" },
      ],
    };
    expect(ratelimitSummary(values)).toBe("5 requests per 1s · per subject + header X-Tenant-Id");
  });

  it("ratelimit with no identifiers", () => {
    const values = {
      ...ratelimit.defaults(),
      type: "ratelimit" as const,
      limit: 10,
      windowMs: 60000,
      identifiers: [],
    };
    expect(ratelimitSummary(values)).toBe("10 requests per 1m · no identifier");
  });

  it("firewall", () => {
    expect(FIREWALL_SUMMARY).toBe("Deny with Forbidden (403)");
  });

  it("openapi", () => {
    expect(OPENAPI_SUMMARY).toBe("Auto-scraped spec");
  });

  it("logging with every option on", () => {
    expect(loggingSummary(logging.defaults())).toBe(
      "request headers, response headers, request body, response body, query",
    );
  });

  it("logging with every option off", () => {
    const values = {
      type: "logging" as const,
      name: "",
      environmentId: "__all__",
      matchConditions: [],
      requestHeaders: false,
      responseHeaders: false,
      requestBody: false,
      responseBody: false,
      query: false,
    };
    expect(loggingSummary(values)).toBe("Default fields only");
  });
});
