import {
  type MatchExpr,
  type Policy,
  fromWirePolicy,
} from "@/lib/collections/deploy/policies.schema";
import { describe, expect, it } from "vitest";
import { fromPolicy, getDefaultValues, toPolicy } from "./index";

const ALL_CONDITIONS: MatchExpr[] = [
  { path: { path: { exact: "/v1/checkout" } } },
  { path: { path: { prefix: "/v1/" } } },
  { path: { path: { regex: "^/v1/.*$" } } },
  { method: { methods: ["GET", "POST"] } },
  { header: { name: "Authorization", value: { prefix: "Bearer " } } },
  { header: { name: "X-Debug", present: true } },
  { queryParam: { name: "debug", present: true } },
  { queryParam: { name: "tenant", value: { exact: "acme" } } },
  { remoteIp: { in: ["203.0.113.0/24", "198.51.100.7"] } },
  { remoteIp: { notIn: ["2001:db8::/32"] } },
];

const WIRE_FIXTURES: Record<string, Policy> = {
  "keyauth with every option": {
    id: "pol_keyauth",
    name: "Checkout auth",
    enabled: true,
    match: ALL_CONDITIONS,
    type: "keyauth",
    keyauth: {
      keyspaces: ["ks_1", "ks_2"],
      locations: [{ header: { name: "Authorization", stripPrefix: "Bearer " } }],
      permissionQuery: "api.read AND api.write",
      ratelimits: [
        { name: "requests" },
        { name: "tokens", cost: 5 },
        { name: "burst", limit: 10, duration: 1000, cost: 2 },
      ],
      credits: 0,
    },
  },
  "keyauth with bearer and query param locations": {
    id: "pol_keyauth_min",
    name: "Partner auth",
    enabled: true,
    match: [],
    type: "keyauth",
    keyauth: {
      keyspaces: ["ks_1"],
      locations: [{ bearer: {} }, { queryParam: { name: "api_key" } }],
      permissionQuery: "",
    },
  },
  ratelimit: {
    id: "pol_ratelimit",
    name: "Burst",
    enabled: true,
    match: [{ path: { path: { regex: "^/v1/share/?$" } } }, { method: { methods: ["POST"] } }],
    type: "ratelimit",
    ratelimit: {
      limit: 100,
      windowMs: 60000,
      identifiers: [
        { remoteIp: {} },
        { header: { name: "X-Tenant-Id" } },
        { authenticatedSubject: {} },
        { path: {} },
        { principalField: { path: "subject" } },
      ],
    },
  },
  firewall: {
    id: "pol_firewall",
    name: "Office only",
    enabled: true,
    match: [{ remoteIp: { notIn: ["198.51.100.0/24"] } }],
    type: "firewall",
    firewall: { action: "ACTION_DENY" },
  },
  openapi: {
    id: "pol_openapi",
    name: "Spec",
    enabled: true,
    match: [],
    type: "openapi",
    openapi: {},
  },
  logging: {
    id: "pol_logging",
    name: "Log everything",
    enabled: true,
    match: [{ header: { name: "X-Debug", present: true } }],
    type: "logging",
    logging: {
      requestHeaders: true,
      responseHeaders: false,
      requestBody: true,
      responseBody: false,
      query: true,
    },
  },
};

function fromApi(policy: Policy): Policy {
  const { type: _type, ...wire } = policy;
  return fromWirePolicy(wire);
}

describe("policy wire format", () => {
  for (const [label, fixture] of Object.entries(WIRE_FIXTURES)) {
    it(`${label}: opening and saving a stored policy writes the same bytes`, () => {
      const saved = toPolicy(fromPolicy(fromApi(fixture)), fixture.id);
      expect(JSON.stringify(saved)).toBe(JSON.stringify(fixture));
    });
  }

  it("writes each kind's defaults as these exact bytes", () => {
    const saved = (["keyauth", "ratelimit", "firewall", "openapi", "logging"] as const).map(
      (type) => toPolicy({ ...getDefaultValues(type), name: "p" }, "pol_1"),
    );
    expect(saved.map((policy) => JSON.stringify(policy))).toEqual([
      '{"id":"pol_1","name":"p","enabled":true,"match":[],"type":"keyauth","keyauth":{"keyspaces":[],"locations":[],"permissionQuery":""}}',
      '{"id":"pol_1","name":"p","enabled":true,"match":[],"type":"ratelimit","ratelimit":{"limit":100,"windowMs":60000,"identifiers":[{"remoteIp":{}}]}}',
      '{"id":"pol_1","name":"p","enabled":true,"match":[],"type":"firewall","firewall":{"action":"ACTION_DENY"}}',
      '{"id":"pol_1","name":"p","enabled":true,"match":[],"type":"openapi","openapi":{}}',
      '{"id":"pol_1","name":"p","enabled":true,"match":[],"type":"logging","logging":{"requestHeaders":true,"responseHeaders":true,"requestBody":true,"responseBody":true,"query":true}}',
    ]);
  });

  it("reads absent logging captures as off and writes them as false", () => {
    const stored: Policy = {
      id: "pol_logging",
      name: "Base log",
      enabled: true,
      type: "logging",
      logging: {},
    };
    expect(JSON.stringify(toPolicy(fromPolicy(fromApi(stored)), stored.id))).toBe(
      '{"id":"pol_logging","name":"Base log","enabled":true,"match":[],"type":"logging","logging":{"requestHeaders":false,"responseHeaders":false,"requestBody":false,"responseBody":false,"query":false}}',
    );
  });

  it("writes a stored keyauth without a permission query as an empty query", () => {
    const stored: Policy = {
      id: "pol_keyauth",
      name: "Auth",
      enabled: true,
      type: "keyauth",
      keyauth: { keyspaces: ["ks_1"] },
    };
    expect(JSON.stringify(toPolicy(fromPolicy(fromApi(stored)), stored.id))).toBe(
      '{"id":"pol_keyauth","name":"Auth","enabled":true,"match":[],"type":"keyauth","keyauth":{"keyspaces":["ks_1"],"locations":[],"permissionQuery":""}}',
    );
  });

  it("writes a stored policy that was off as on", () => {
    const stored: Policy = { ...WIRE_FIXTURES.firewall, enabled: false };
    expect(toPolicy(fromPolicy(fromApi(stored)), stored.id).enabled).toBe(true);
  });
});
