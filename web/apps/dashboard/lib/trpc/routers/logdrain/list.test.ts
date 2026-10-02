import { initTRPC } from "@trpc/server";
import { expect, it, vi } from "vitest";
import { type LogdrainConfig, encodeLogdrainConfig } from "./config";
import { listLogdrains } from "./list";

const selectRows = vi.hoisted(() => vi.fn());
vi.mock("@/lib/db", () => ({
  db: { select: () => ({ from: () => ({ where: selectRows }) }) },
  eq: vi.fn(),
  schema: { logdrains: {} },
}));
vi.mock("../../trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  return {
    workspaceProcedure: initTRPC.context<{ workspace: { id: string } }>().create().procedure,
  };
});

it("lists all five streams when a rate-limit drain shares the workspace", async () => {
  const streams: LogdrainConfig["stream"][] = [
    { kind: "audit_logs", eventTypes: ["key.create"] },
    { kind: "ratelimits", namespaceIds: ["payments"], passed: [false] },
    { kind: "key_verifications", keySpaceIds: ["production"], outcomes: ["VALID"] },
    {
      kind: "gateway_requests",
      projectIds: ["store"],
      appIds: [],
      environmentIds: [],
      statusClasses: [5],
    },
    {
      kind: "runtime_logs",
      projectIds: [],
      appIds: ["worker"],
      environmentIds: [],
      severities: ["warn"],
    },
  ];
  selectRows.mockResolvedValue(
    streams.map((stream) => ({
      id: stream.kind,
      stream: stream.kind,
      config: encodeLogdrainConfig({
        stream,
        kind: "axiom",
        dataset: "logs",
        encryptedToken: "private-ciphertext",
      }),
    })),
  );
  const t = initTRPC.context<{ workspace: { id: string } }>().create();
  const caller = t.router({ list: listLogdrains }).createCaller({ workspace: { id: "workspace" } });
  const drains = await caller.list();
  expect(drains.map(({ stream }) => stream)).toEqual([
    "audit_logs",
    "ratelimits",
    "key_verifications",
    "gateway_requests",
    "runtime_logs",
  ]);
  expect(drains[1]).toMatchObject({ namespaceIds: ["payments"], passed: [false] });
  expect(JSON.stringify(drains)).not.toContain("private-ciphertext");
});
