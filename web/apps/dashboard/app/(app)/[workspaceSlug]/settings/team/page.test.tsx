import { cleanup, render, screen } from "@testing-library/react";
import type React from "react";
import { afterEach, beforeEach, describe, expect, it, vi } from "vitest";

const mocks = vi.hoisted(() => ({
  authProvider: "workos" as "workos" | "local",
  getAuth: vi.fn(),
  findWorkspace: vi.fn(),
  workosAuthEnv: vi.fn(),
}));

vi.mock("@/lib/auth", () => ({
  getAuth: mocks.getAuth,
}));

vi.mock("@/lib/db", () => ({
  db: {
    query: {
      workspaces: {
        findFirst: mocks.findWorkspace,
      },
    },
  },
}));

vi.mock("@/lib/env", () => ({
  env: () => ({ AUTH_PROVIDER: mocks.authProvider }),
  workosAuthEnv: mocks.workosAuthEnv,
}));

vi.mock("@unkey/ui", () => {
  const Wrapper = ({ children }: { children?: React.ReactNode }) => <div>{children}</div>;

  return {
    PageBody: Wrapper,
    PageContainer: Wrapper,
    PageHeader: Wrapper,
    PageHeaderContent: Wrapper,
    PageHeaderTitle: ({ children }: { children?: React.ReactNode }) => <h1>{children}</h1>,
  };
});

vi.mock("./client", () => ({
  TeamPageClient: ({ team }: { team: boolean }) => (
    <div data-testid="local-team">{team ? "enabled" : "disabled"}</div>
  ),
}));

vi.mock("./managed-team", () => ({
  ManagedTeam: ({ team }: { team: boolean }) => (
    <div data-testid="managed-team">{team ? "enabled" : "disabled"}</div>
  ),
}));

import SettingTeamPage from "./page";

afterEach(cleanup);

describe("Team page provider split", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.authProvider = "workos";
    mocks.getAuth.mockResolvedValue({ orgId: "org_123" });
    mocks.findWorkspace.mockResolvedValue({
      id: "ws_123",
      quotas: { team: true },
    });
  });

  it("loads WorkOS user management in managed-auth mode", async () => {
    render(await SettingTeamPage());

    expect(screen.getByTestId("managed-team").textContent).toBe("enabled");
    expect(screen.queryByTestId("local-team")).toBeNull();
    expect(mocks.workosAuthEnv).toHaveBeenCalledOnce();
  });

  it("keeps the existing Team client isolated to local mode", async () => {
    mocks.authProvider = "local";

    render(await SettingTeamPage());

    expect(screen.getByTestId("local-team").textContent).toBe("enabled");
    expect(screen.queryByTestId("managed-team")).toBeNull();
    expect(mocks.workosAuthEnv).not.toHaveBeenCalled();
  });
});
