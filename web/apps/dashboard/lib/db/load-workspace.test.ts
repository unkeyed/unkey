import { beforeEach, expect, it, vi } from "vitest";
import { loadWorkspace } from "./load-workspace";

const mocks = vi.hoisted(() => ({ findFirst: vi.fn() }));

vi.mock("@/lib/db", () => ({
  db: { query: { workspaces: { findFirst: mocks.findFirst } } },
}));

beforeEach(() => vi.resetAllMocks());

it("uses database values before code fallbacks and returns only requested flags", async () => {
  mocks.findFirst.mockResolvedValue({
    id: "ws_owned",
    billing: null,
    billingSubscriptions: [],
    flags: { disabled: 0, enabled: 1, unrequested: 1 },
  });

  const workspace = await loadWorkspace("org_owned", [
    { slug: "disabled", defaultValue: true },
    { slug: "enabled", defaultValue: false },
    { slug: "missing", defaultValue: true },
    { slug: "constructor", defaultValue: false },
  ]);

  expect(workspace?.flags).toEqual({
    disabled: false,
    enabled: true,
    missing: true,
    constructor: false,
  });
});
