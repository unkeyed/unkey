import { TRPCError } from "@trpc/server";
import { describe, expect, it, vi } from "vitest";
import { applyHttpHeaderUpdates } from "./update";

vi.mock("@/lib/audit", () => ({ insertAuditLogs: vi.fn() }));
vi.mock("@/lib/db", () => ({ and: vi.fn(), db: {}, eq: vi.fn(), schema: {} }));
vi.mock("@/lib/vault-client", () => ({ createVaultClient: vi.fn(() => ({})) }));
vi.mock("../../trpc", () => ({
  workspaceProcedure: {
    input: vi.fn(() => ({ mutation: vi.fn(() => ({})) })),
  },
}));

describe("applyHttpHeaderUpdates", () => {
  const existing = [
    { name: "Authorization", encryptedValue: "encrypted-old-token" },
    { name: "X-Customer", encryptedValue: "encrypted-customer" },
  ];

  it("preserves, replaces, adds, and removes headers without plaintext values", () => {
    expect(
      applyHttpHeaderUpdates({
        existing,
        updates: [
          { mode: "preserve", name: "Authorization" },
          { mode: "set", name: "X-Source", value: "source" },
        ],
        encrypted: [{ name: "X-Source", encryptedValue: "encrypted-source" }],
      }),
    ).toEqual([
      { name: "Authorization", encryptedValue: "encrypted-old-token" },
      { name: "X-Source", encryptedValue: "encrypted-source" },
    ]);
  });

  it("matches preserved names without case sensitivity", () => {
    expect(
      applyHttpHeaderUpdates({
        existing,
        updates: [{ mode: "preserve", name: "authorization" }],
        encrypted: [],
      }),
    ).toEqual([{ name: "Authorization", encryptedValue: "encrypted-old-token" }]);
  });

  it("keeps the stored name when it replaces a value", () => {
    expect(
      applyHttpHeaderUpdates({
        existing,
        updates: [{ mode: "set", name: "authorization", value: "replacement" }],
        encrypted: [{ name: "authorization", encryptedValue: "encrypted-replacement" }],
      }),
    ).toEqual([{ name: "Authorization", encryptedValue: "encrypted-replacement" }]);
  });

  it("rejects an unknown header preservation request", () => {
    expect(() =>
      applyHttpHeaderUpdates({
        existing,
        updates: [{ mode: "preserve", name: "X-Unknown" }],
        encrypted: [],
      }),
    ).toThrow(TRPCError);
  });
});
