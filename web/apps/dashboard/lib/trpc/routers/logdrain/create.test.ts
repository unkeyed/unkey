import { describe, expect, it, vi } from "vitest";
import { decodeLogdrainConfig } from "./config";
import "./create";

const procedure = vi.hoisted(() => ({ parse: (_input: unknown): unknown => undefined }));
const saved = vi.hoisted(() =>
  vi.fn<
    [{ stream: string; config: Uint8Array; createdAt: number; committedOffsetInsertedAt: number }],
    void
  >(),
);
const mutation = vi.hoisted(() => ({
  run: async (_input: unknown): Promise<unknown> => undefined,
}));
vi.mock("@/lib/audit", () => ({ insertAuditLogs: vi.fn() }));
vi.mock("@/lib/vault-client", () => ({
  createVaultClient: () => ({ encrypt: async () => ({ encrypted: "ciphertext" }) }),
}));
vi.mock("@/lib/db", () => ({
  db: {
    transaction: async (run: (tx: unknown) => Promise<void>) =>
      run({ insert: () => ({ values: saved }) }),
  },
  schema: { logdrains: {} },
}));
vi.mock("../../trpc", () => ({
  workspaceProcedure: {
    input: (schema: { parse: (input: unknown) => unknown }) => {
      procedure.parse = schema.parse.bind(schema);
      return {
        mutation: (run: (options: unknown) => Promise<unknown>) => {
          mutation.run = (input) =>
            run({
              input: procedure.parse(input),
              ctx: {
                workspace: { id: "ws_test" },
                user: { id: "user_test" },
                audit: { location: "", userAgent: "test" },
              },
            });
          return {};
        },
      };
    },
  },
}));

describe("create gateway log drain", () => {
  it.each(["http", "axiom"])("creates runtime drains without backfill for %s", async (kind) => {
    saved.mockClear();
    await mutation.run({
      name: "Runtime",
      stream: "runtime_logs",
      severities: ["error", "warn"],
      projectIds: ["project"],
      appIds: ["app"],
      environmentIds: [],
      kind,
      config:
        kind === "http" ? { url: "https://example.com" } : { dataset: "runtime", token: "secret" },
    });
    const row = saved.mock.calls[0]?.[0];
    if (!row) {
      throw new Error("No drain was persisted");
    }
    expect(row.stream).toBe("runtime_logs");
    expect(row.committedOffsetInsertedAt).toBe(row.createdAt);
    expect(decodeLogdrainConfig(row.config).stream).toEqual({
      kind: "runtime_logs",
      severities: ["error", "warn"],
      projectIds: ["project"],
      appIds: ["app"],
      environmentIds: [],
    });
  });

  it.each(["http", "axiom"])(
    "persists typed statuses and starts at creation time for %s",
    async (kind) => {
      saved.mockClear();
      const before = Date.now();
      await mutation.run({
        name: "Requests",
        stream: "gateway_requests",
        statusClasses: [4, 5],
        projectIds: ["project"],
        appIds: ["app"],
        environmentIds: ["env"],
        kind,
        config:
          kind === "http"
            ? { url: "https://example.com", format: "ndjson" }
            : { dataset: "requests", token: "secret" },
      });
      const row = saved.mock.calls[0]?.[0];
      if (!row) {
        throw new Error("No drain was persisted");
      }
      expect(row.stream).toBe("gateway_requests");
      expect(row.committedOffsetInsertedAt).toBe(row.createdAt);
      expect(row.createdAt).toBeGreaterThanOrEqual(before);
      expect(decodeLogdrainConfig(row.config).stream).toEqual({
        kind: "gateway_requests",
        statusClasses: [4, 5],
        projectIds: ["project"],
        appIds: ["app"],
        environmentIds: ["env"],
      });
    },
  );

  it("rejects filters for a different stream", () => {
    for (const filters of [{ eventTypes: [] }, { outcomes: [] }, { keySpaceIds: [] }]) {
      expect(() =>
        procedure.parse({
          name: "Requests",
          stream: "gateway_requests",
          ...filters,
          kind: "http",
          config: { url: "https://example.com" },
        }),
      ).toThrow();
    }
    for (const stream of ["audit_logs", "key_verifications"]) {
      for (const filters of [
        { statusClasses: [] },
        { projectIds: [] },
        { appIds: [] },
        { environmentIds: [] },
      ]) {
        expect(() =>
          procedure.parse({
            name: "Other",
            stream,
            ...filters,
            kind: "http",
            config: { url: "https://example.com" },
          }),
        ).toThrow();
      }
    }
  });
});
