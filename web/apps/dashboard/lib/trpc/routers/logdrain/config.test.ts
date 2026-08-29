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
      url: "https://example.com/logs",
      format: "ndjson",
      headers: [
        { name: "Authorization", encryptedValue: "encrypted-authorization" },
        { name: "X-Customer", encryptedValue: "encrypted-customer" },
      ],
    },
    {
      kind: "axiom",
      dataset: "audit-logs",
      encryptedToken: "encrypted-axiom-token",
    },
  ];

  for (const config of configs) {
    it(`round trips ${config.kind} config`, () => {
      expect(decodeLogdrainConfig(encodeLogdrainConfig(config))).toEqual(config);
    });
  }

  it("rejects config without a provider", () => {
    expect(() => decodeLogdrainConfig(new Uint8Array())).toThrow("provider is not set");
  });

  it("encrypts each HTTP header value and preserves its name", async () => {
    vault.encryptBulk.mockResolvedValue({
      items: {
        "0": { encrypted: "encrypted-authorization" },
        "1": { encrypted: "encrypted-customer" },
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
      items: { "0": "Bearer token", "1": "customer-value" },
    });
  });
});
