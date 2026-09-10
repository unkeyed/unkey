import { ConfigSchema } from "@/gen/proto/logdrain/v1/config_pb";
import { fromBinary } from "@bufbuild/protobuf";
import { describe, expect, it, vi } from "vitest";
import {
  type LogdrainConfig,
  decodeLogdrainConfig,
  encodeLogdrainConfig,
  encryptHttpHeaders,
} from "./config";

const vault = vi.hoisted(() => ({ encryptBulk: vi.fn() }));
vi.mock("@/lib/vault-client", () => ({ createVaultClient: () => vault }));

describe("log drain protobuf config", () => {
  const configs: LogdrainConfig[] = [
    {
      kind: "http",
      stream: { kind: "audit_logs", eventTypes: ["key.create", "future.event"] },
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
  ];

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
