import { initTRPC } from "@trpc/server";
import { beforeEach, expect, it, vi } from "vitest";
import { getCurrentWorkspace } from "./getCurrent";

const mocks = vi.hoisted(() => ({ findFirst: vi.fn() }));

vi.mock("@/lib/db", () => ({
  db: { query: { workspaces: { findFirst: mocks.findFirst } } },
}));

vi.mock("../../trpc", async () => {
  const { initTRPC } = await import("@trpc/server");
  return { protectedProcedure: initTRPC.create().procedure };
});

const router = initTRPC
  .context<{
    workspace?: { id: string; flags: Record<string, boolean> };
    tenant: { id: string } | null;
  }>()
  .create()
  .router({ getCurrent: getCurrentWorkspace });

beforeEach(() => {
  vi.resetAllMocks();
});

it("returns flags already loaded with the workspace", async () => {
  const workspace = { id: "ws_owned", flags: { preview: false, routing: true } };
  const result = await router
    .createCaller({ workspace, tenant: { id: "org_session" } })
    .getCurrent();
  expect(result).toEqual(workspace);
});

it("returns flags from the fallback workspace query", async () => {
  mocks.findFirst.mockResolvedValue({ id: "ws_fallback", flags: { preview: true } });
  const result = await router.createCaller({ tenant: { id: "org_session" } }).getCurrent();
  expect(result).toMatchObject({ id: "ws_fallback", flags: { preview: true } });
});
