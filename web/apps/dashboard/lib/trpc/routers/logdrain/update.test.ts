import { TRPCError, type inferProcedureInput } from "@trpc/server";
import { describe, expect, it, vi } from "vitest";
import { decodeLogdrainConfig, encodeLogdrainConfig } from "./config";
import { applyHttpHeaderUpdates, type updateLogdrain } from "./update";

type Mutation = (options: {
  input: inferProcedureInput<typeof updateLogdrain>;
  ctx: {
    workspace: { id: string };
    user: { id: string };
    audit: { location: string; userAgent: string };
  };
}) => Promise<unknown>;

const procedure = vi.hoisted(() => ({
  safeParse: (_input: unknown): { success: boolean } => {
    throw new Error("Input schema was not registered");
  },
  mutate: async (_options: Parameters<Mutation>[0]): Promise<unknown> => {
    throw new Error("Mutation was not registered");
  },
}));

const database = vi.hoisted(() => {
  const read = vi.fn();
  const write = vi
    .fn<[Record<string, unknown> & { config: Uint8Array }], { where: ReturnType<typeof vi.fn> }>()
    .mockReturnValue({ where: vi.fn() });
  const tx = {
    select: () => ({ from: () => ({ where: () => ({ for: read }) }) }),
    update: () => ({ set: write }),
  };
  return { read, write, transaction: async (run: (value: typeof tx) => Promise<void>) => run(tx) };
});

vi.mock("@/lib/audit", () => ({ insertAuditLogs: vi.fn() }));
vi.mock("@/lib/db", () => ({
  and: vi.fn(),
  db: database,
  eq: vi.fn(),
  schema: {
    logdrains: {
      id: "id",
      name: "name",
      config: "config",
      status: "status",
      workspaceId: "workspaceId",
    },
  },
}));
vi.mock("@/lib/vault-client", () => ({ createVaultClient: vi.fn(() => ({})) }));
vi.mock("../../trpc", () => ({
  workspaceProcedure: {
    input: vi.fn((schema: { safeParse: (input: unknown) => { success: boolean } }) => {
      procedure.safeParse = (input) => schema.safeParse(input);
      return {
        mutation: (mutation: Mutation) => {
          procedure.mutate = mutation;
          return {};
        },
      };
    }),
  },
}));

describe("updateLogdrain input", () => {
  it("accepts verification outcomes and rejects audit event names as outcomes", () => {
    expect(procedure.safeParse({ id: "ld_test", outcomes: ["RATE_LIMITED"] }).success).toBe(true);
    expect(procedure.safeParse({ id: "ld_test", outcomes: ["key.create"] }).success).toBe(false);
  });

  it("accepts a filter-only update including future event names", () => {
    expect(
      procedure.safeParse({ id: "ld_test", eventTypes: ["key.create", "future.event"] }).success,
    ).toBe(true);
  });

  it.each([
    {
      field: "HTTP URL",
      destination: { kind: "http", config: { url: "https://new.example.com" } },
    },
    { field: "HTTP format", destination: { kind: "http", config: { format: "ndjson" } } },
    {
      field: "HTTP headers",
      destination: {
        kind: "http",
        config: { headers: [{ mode: "preserve", name: "Authorization" }] },
      },
    },
    { field: "Axiom dataset", destination: { kind: "axiom", config: { dataset: "new-dataset" } } },
    { field: "Axiom token", destination: { kind: "axiom", config: { token: "new-token" } } },
  ])("accepts a partial $field update without stale sibling fields", ({ destination }) => {
    expect(procedure.safeParse({ id: "ld_test", destination }).success).toBe(true);
  });

  it.each(["http", "axiom"])("rejects an empty %s destination update", (kind) => {
    expect(procedure.safeParse({ id: "ld_test", destination: { kind, config: {} } }).success).toBe(
      false,
    );
  });
});

describe("updateLogdrain event filters", () => {
  it.each([{ outcomes: [] }, { outcomes: ["EXPIRED" as const] }])(
    "updates verification outcomes without replay and rejects audit filters",
    async ({ outcomes }) => {
      database.read.mockResolvedValue([
        {
          id: "ld_test",
          name: "Verifications",
          status: "running",
          config: encodeLogdrainConfig({
            kind: "http",
            stream: { kind: "key_verifications", outcomes: ["VALID"] },
            url: "https://example.com",
            format: "json",
            headers: [],
          }),
        },
      ]);
      const ctx = {
        workspace: { id: "ws_test" },
        user: { id: "user_test" },
        audit: { location: "", userAgent: "test" },
      };
      database.write.mockClear();
      await procedure.mutate({ input: { id: "ld_test", outcomes }, ctx });
      const saved = database.write.mock.calls[0]?.[0];
      if (!saved) {
        throw new Error("No drain update was persisted");
      }
      expect(decodeLogdrainConfig(saved.config).stream).toEqual({
        kind: "key_verifications",
        outcomes,
      });
      expect(saved).toMatchObject({ leaseExpiresAt: 0, consecutiveFailures: 0, nextAttemptAt: 0 });
      expect(saved).not.toHaveProperty("committedOffsetInsertedAt");
      expect(saved).not.toHaveProperty("committedOffsetEventId");
      database.write.mockClear();
      await expect(
        procedure.mutate({ input: { id: "ld_test", eventTypes: [] }, ctx }),
      ).rejects.toMatchObject({ code: "BAD_REQUEST" });
      expect(database.write).not.toHaveBeenCalled();
    },
  );

  it.each([
    {
      name: "preserves filters on a destination edit",
      eventTypes: undefined,
      destination: { kind: "http" as const, config: { format: "ndjson" as const } },
      expected: ["key.create"],
    },
    {
      name: "clears filters without changing the destination",
      eventTypes: [],
      destination: undefined,
      expected: [],
    },
    {
      name: "replaces filters without changing the destination",
      eventTypes: ["key.delete"],
      destination: undefined,
      expected: ["key.delete"],
    },
  ])(
    "$name and fences in-flight work without resetting the cursor",
    async ({ eventTypes, destination, expected }) => {
      database.read.mockResolvedValue([
        {
          id: "ld_test",
          name: "Audit",
          status: "running",
          config: encodeLogdrainConfig({
            kind: "http",
            stream: { kind: "audit_logs", eventTypes: ["key.create"] },
            url: "https://example.com",
            format: "json",
            headers: [],
          }),
        },
      ]);
      database.write.mockClear();
      await procedure.mutate({
        input: { id: "ld_test", eventTypes, destination },
        ctx: {
          workspace: { id: "ws_test" },
          user: { id: "user_test" },
          audit: { location: "127.0.0.1", userAgent: "test" },
        },
      });
      expect(database.write).toHaveBeenCalledOnce();
      const saved = database.write.mock.calls[0]?.[0];
      if (!saved) {
        throw new Error("No drain update was persisted");
      }
      expect(decodeLogdrainConfig(saved.config)).toEqual({
        kind: "http",
        stream: { kind: "audit_logs", eventTypes: expected },
        url: "https://example.com",
        format: destination ? "ndjson" : "json",
        headers: [],
      });
      expect(saved).toMatchObject({ leaseExpiresAt: 0, consecutiveFailures: 0, nextAttemptAt: 0 });
      expect(saved).not.toHaveProperty("committedOffsetInsertedAt");
      expect(saved).not.toHaveProperty("committedOffsetEventId");
    },
  );
});

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
