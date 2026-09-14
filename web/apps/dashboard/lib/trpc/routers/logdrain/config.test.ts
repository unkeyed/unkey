import { ConfigSchema } from "@/gen/proto/logdrain/v1/config_pb";
import { create, fromBinary, toBinary } from "@bufbuild/protobuf";
import { describe, expect, it, vi } from "vitest";
import {
  type LogdrainConfig,
  decodeLogdrainConfig,
  encodeLogdrainConfig,
  encryptHttpHeaders,
  toPublicLogdrainConfig,
} from "./config";

const vault = vi.hoisted(() => ({ encryptBulk: vi.fn() }));
vi.mock("@/lib/vault-client", () => ({ createVaultClient: () => vault }));

describe("log drain protobuf config", () => {
  it("preserves a per-drain batch size through a config round trip", () => {
    const config = {
      kind: "axiom" as const,
      dataset: "decisions",
      encryptedToken: "secret",
      batchSize: 137,
      stream: { kind: "audit_logs" as const, eventTypes: [] },
    };
    expect(decodeLogdrainConfig(encodeLogdrainConfig(config))).toEqual(config);
  });

  it("round trips exact rate-limit filters and hides destination credentials", () => {
    const config = {
      kind: "axiom" as const,
      dataset: "decisions",
      encryptedToken: "secret",
      stream: {
        kind: "ratelimits" as const,
        namespaceIds: ["ns"],
        passed: [false],
      },
    };
    const decoded = decodeLogdrainConfig(encodeLogdrainConfig(config));
    expect(decoded).toEqual(config);
    const publicConfig = toPublicLogdrainConfig(decoded);
    expect(publicConfig).toMatchObject({
      stream: "ratelimits",
      namespaceIds: ["ns"],
      passed: [false],
    });
    expect(JSON.stringify(publicConfig)).not.toContain("secret");
  });

  it("projects stream filters independently of the destination without exposing credentials", () => {
    const stream = {
      kind: "gateway_requests" as const,
      statusClasses: [4, 5],
      projectIds: ["project"],
      appIds: ["app"],
      environmentIds: ["env"],
    };
    const filters = {
      stream: "gateway_requests",
      eventTypes: [],
      outcomes: [],
      keySpaceIds: [],
      statusClasses: [4, 5],
      severities: [],
      projectIds: ["project"],
      appIds: ["app"],
      environmentIds: ["env"],
    };
    expect(
      toPublicLogdrainConfig({
        kind: "http",
        stream,
        url: "https://example.com",
        format: "ndjson",
        headers: [{ name: "Authorization", encryptedValue: "secret" }],
      }),
    ).toEqual({
      ...filters,
      kind: "http",
      config: { url: "https://example.com", format: "ndjson", headers: ["Authorization"] },
    });
    expect(
      toPublicLogdrainConfig({
        kind: "axiom",
        stream,
        dataset: "requests",
        encryptedToken: "secret",
      }),
    ).toEqual({ ...filters, kind: "axiom", config: { dataset: "requests" } });
  });

  it("keeps audit and verification public fields unchanged", () => {
    expect(
      toPublicLogdrainConfig({
        kind: "axiom",
        dataset: "audit",
        encryptedToken: "secret",
        stream: { kind: "audit_logs", eventTypes: ["key.create"] },
      }),
    ).toEqual({
      kind: "axiom",
      config: { dataset: "audit" },
      stream: "audit_logs",
      eventTypes: ["key.create"],
      outcomes: [],
      keySpaceIds: [],
      statusClasses: [],
      severities: [],
      projectIds: [],
      appIds: [],
      environmentIds: [],
    });
    expect(
      toPublicLogdrainConfig({
        kind: "axiom",
        dataset: "verification",
        encryptedToken: "secret",
        stream: {
          kind: "key_verifications",
          outcomes: ["RATE_LIMITED"],
          keySpaceIds: ["ks_primary"],
        },
      }),
    ).toEqual({
      kind: "axiom",
      config: { dataset: "verification" },
      stream: "key_verifications",
      eventTypes: [],
      outcomes: ["RATE_LIMITED"],
      keySpaceIds: ["ks_primary"],
      statusClasses: [],
      severities: [],
      projectIds: [],
      appIds: [],
      environmentIds: [],
    });
  });

  it("round trips gateway resource and status class filters", () => {
    const config = {
      kind: "axiom" as const,
      stream: {
        kind: "gateway_requests" as const,
        statusClasses: [4, 5],
        projectIds: ["project"],
        appIds: ["app"],
        environmentIds: ["env"],
      },
      dataset: "requests",
      encryptedToken: "ciphertext",
    };
    const encoded = encodeLogdrainConfig(config);
    expect(fromBinary(ConfigSchema, encoded).stream).toMatchObject({
      case: "gatewayRequests",
      value: {
        statusClasses: [4, 5],
        projectIds: ["project"],
        appIds: ["app"],
        environmentIds: ["env"],
      },
    });
    expect(decodeLogdrainConfig(encoded)).toEqual(config);
  });

  const configs = [
    {
      kind: "http",
      stream: {
        kind: "audit_logs",
        eventTypes: ["key.create", "future.event"],
      },
      url: "https://example.com/logs",
      format: "ndjson",
      headers: [
        { name: "Authorization", encryptedValue: "encrypted-authorization" },
        { name: "X-Customer", encryptedValue: "encrypted-customer" },
      ],
    },
    {
      kind: "axiom",
      stream: { kind: "audit_logs", eventTypes: ["portal.create"] },
      dataset: "audit-logs",
      encryptedToken: "encrypted-axiom-token",
    },
  ] satisfies LogdrainConfig[];

  for (const config of configs) {
    it(`round trips ${config.kind} config`, () => {
      const encoded = encodeLogdrainConfig(config);
      expect(fromBinary(ConfigSchema, encoded).stream).toMatchObject({
        case: "auditLogs",
        value: { eventTypes: config.stream.eventTypes },
      });
      expect(decodeLogdrainConfig(encoded)).toEqual(config);
    });
  }

  it("rejects config without a provider", () => {
    expect(() => decodeLogdrainConfig(new Uint8Array())).toThrow("provider is not set");
  });

  it("round trips runtime severities and resources", () => {
    const config = {
      kind: "axiom" as const,
      stream: {
        kind: "runtime_logs" as const,
        severities: ["warn", "error"],
        projectIds: ["project"],
        appIds: ["app", "other-app"],
        environmentIds: [],
      },
      dataset: "runtime",
      encryptedToken: "ciphertext",
    };
    expect(decodeLogdrainConfig(encodeLogdrainConfig(config))).toEqual(config);
  });

  it("round trips verification outcomes independently of the destination", () => {
    const config = {
      kind: "axiom" as const,
      stream: {
        kind: "key_verifications" as const,
        outcomes: ["RATE_LIMITED", "EXPIRED"],
        keySpaceIds: ["ks_primary", "ks_secondary"],
      },
      dataset: "verifications",
      encryptedToken: "ciphertext",
    };
    const encoded = encodeLogdrainConfig(config);
    expect(fromBinary(ConfigSchema, encoded).stream).toMatchObject({
      case: "keyVerifications",
      value: {
        outcomes: ["RATE_LIMITED", "EXPIRED"],
        keySpaceIds: ["ks_primary", "ks_secondary"],
      },
    });
    expect(decodeLogdrainConfig(encoded)).toEqual(config);
  });

  it("keeps older verification configs unfiltered by keyspace", () => {
    const encoded = toBinary(
      ConfigSchema,
      create(ConfigSchema, {
        stream: { case: "keyVerifications", value: { outcomes: ["VALID"] } },
        destination: { case: "axiom", value: { dataset: "verifications" } },
      }),
    );
    expect(decodeLogdrainConfig(encoded).stream).toEqual({
      kind: "key_verifications",
      outcomes: ["VALID"],
      keySpaceIds: [],
    });
  });

  it("decodes configs without event types as all events", () => {
    const legacy = Buffer.from("12070a056175646974", "hex");
    expect(decodeLogdrainConfig(legacy)).toEqual({
      kind: "axiom",
      stream: { kind: "audit_logs", eventTypes: [] },
      dataset: "audit",
      encryptedToken: "",
    });
  });

  it("encrypts each HTTP header value and preserves its name", async () => {
    vault.encryptBulk.mockResolvedValue({
      items: {
        Authorization: { encrypted: "encrypted-authorization" },
        "X-Customer": { encrypted: "encrypted-customer" },
      },
    });

    await expect(
      encryptHttpHeaders("ws_123", {
        "X-Customer": "customer-value",
        Authorization: "Bearer token",
      }),
    ).resolves.toEqual([
      { name: "Authorization", encryptedValue: "encrypted-authorization" },
      { name: "X-Customer", encryptedValue: "encrypted-customer" },
    ]);
    expect(vault.encryptBulk).toHaveBeenCalledWith({
      keyring: "ws_123",
      items: {
        "X-Customer": "customer-value",
        Authorization: "Bearer token",
      },
    });
  });
});
