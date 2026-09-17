import { render } from "@testing-library/react";
import { beforeEach, describe, expect, it, vi } from "vitest";
import { WorkspaceProvider } from "./workspace-provider";

const mocks = vi.hoisted(() => ({
  pathname: "/apis",
  userRefetch: vi.fn(),
  workspaceRefetch: vi.fn(),
  userQuery: {
    data: undefined,
    isLoading: false,
    error: {
      data: {
        code: "UNAUTHORIZED",
      },
    },
  },
  workspaceQuery: {
    data: undefined,
    isLoading: false,
    error: null,
  },
}));

vi.mock("next/navigation", () => ({
  usePathname: () => mocks.pathname,
}));

vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    user: {
      getCurrentUser: {
        useQuery: () => ({
          ...mocks.userQuery,
          refetch: mocks.userRefetch,
        }),
      },
    },
    workspace: {
      getCurrent: {
        useQuery: () => ({
          ...mocks.workspaceQuery,
          refetch: mocks.workspaceRefetch,
        }),
      },
    },
  },
}));

describe("WorkspaceProvider", () => {
  beforeEach(() => {
    vi.clearAllMocks();
    mocks.pathname = "/apis";
  });

  it("does not refetch the current user after an unauthorized response", () => {
    render(
      <WorkspaceProvider>
        <div>Dashboard</div>
      </WorkspaceProvider>,
    );

    expect(mocks.userRefetch).not.toHaveBeenCalled();
  });
});
