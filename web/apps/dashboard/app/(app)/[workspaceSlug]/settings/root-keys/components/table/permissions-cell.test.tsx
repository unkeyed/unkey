import { cleanup, render, screen } from "@testing-library/react";
import type { ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { PermissionsCell } from "./permissions-cell";

// @unkey/ui declares react as a dependency rather than a peer, so under vitest
// its primitives run against a second React copy and every hook call throws.
vi.mock("@unkey/ui", () => ({
  InfoTooltip: ({ children }: { children: ReactNode }) => <>{children}</>,
}));

afterEach(cleanup);

const permissions = (...names: string[]) => names.map((name) => ({ name }));

describe("PermissionsCell", () => {
  it("renders a urn grant as a collection noun and action", () => {
    render(
      <PermissionsCell
        permissions={permissions("unkey:v1:ws_ktm9pDvEcVca:ratelimits/namespaces/*#read_namespace")}
      />,
    );

    expect(screen.getByText("Namespaces · Read")).toBeDefined();
    expect(screen.queryByText(/unkey:v1:/)).toBeNull();
    expect(screen.queryByText(/ws_ktm9pDvEcVca/)).toBeNull();
  });

  it("keeps an action that does not repeat its collection", () => {
    render(
      <PermissionsCell
        permissions={permissions(
          "unkey:v1:ws_ktm9pDvEcVca:ratelimits/namespaces/ns_1/overrides/*#set_override",
        )}
      />,
    );

    expect(screen.getByText("Overrides · Set")).toBeDefined();
  });

  it("humanises a legacy grant", () => {
    render(<PermissionsCell permissions={permissions("api.*.create_api")} />);

    expect(screen.getByText("Create API")).toBeDefined();
  });

  it("hides grants past the first two behind an overflow count", () => {
    render(
      <PermissionsCell
        permissions={permissions(
          "unkey:v1:ws_1:identities/*#read_identity",
          "unkey:v1:ws_1:rbac/roles/*#create_role",
          "unkey:v1:ws_1:keyspaces/ks_1/keys/*#delete_key",
        )}
      />,
    );

    expect(screen.getByText("Identities · Read identity, Roles · Create")).toBeDefined();
    expect(screen.getByText("+1")).toBeDefined();
  });

  it("renders a dash when there are no grants", () => {
    render(<PermissionsCell permissions={[]} />);

    expect(screen.getByText("—")).toBeDefined();
  });
});
