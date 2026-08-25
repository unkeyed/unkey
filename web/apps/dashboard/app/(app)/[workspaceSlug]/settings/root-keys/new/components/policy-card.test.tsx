import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ALL_INSTANCES, newPolicy, setRowActions } from "../lib/policy";
import { PolicyCard } from "./policy-card";

vi.mock("@unkey/icons", () => ({
  ChevronRight: () => null,
  CircleInfo: () => null,
  Check: () => null,
  ChevronDown: () => null,
  ChevronExpandY: () => null,
  Trash: () => null,
  XMark: () => null,
}));

vi.mock("@unkey/ui", () => ({
  Button: ({ children, ...props }: ButtonHTMLAttributes<HTMLButtonElement>) => (
    <button type="button" {...props}>
      {children}
    </button>
  ),
  Checkbox: ({
    id,
    checked,
    onCheckedChange,
  }: {
    id?: string;
    checked?: boolean;
    onCheckedChange: (checked: boolean) => void;
  }) => (
    <input
      id={id}
      type="checkbox"
      checked={checked}
      onChange={(event) => onCheckedChange(event.currentTarget.checked)}
    />
  ),
  Input: (props: InputHTMLAttributes<HTMLInputElement>) => <input {...props} />,
  InfoTooltip: ({ children }: { children: ReactNode }) => children,
  Select: ({
    value,
    items,
    onValueChange,
  }: {
    value: string;
    items: { value: string; label: string }[];
    onValueChange: (value: string) => void;
  }) => (
    <select
      aria-label="Resource type"
      value={value}
      onChange={(event) => onValueChange(event.currentTarget.value)}
    >
      {items.map((item) => (
        <option key={item.value} value={item.value}>
          {item.label}
        </option>
      ))}
    </select>
  ),
  SelectContent: () => null,
  SelectItem: () => null,
  SelectTrigger: () => null,
  SelectValue: () => null,
}));

vi.mock("@/lib/trpc/client", () => ({
  trpc: {
    deploy: {
      project: {
        list: {
          useQuery: () => ({
            data: [
              {
                id: "proj_1",
                name: "web platform",
                apps: [{ id: "app_1", name: "site" }],
              },
            ],
            isLoading: false,
          }),
        },
      },
      environment: {
        listAll: {
          useQuery: () => ({
            data: [{ id: "env_1", name: "production", projectId: "proj_1", appId: "app_1" }],
            isLoading: false,
          }),
        },
      },
      environmentSettings: {
        getAvailableKeyspaces: {
          useQuery: () => ({
            data: { ks_1: { id: "ks_1", api: { name: "payments" } } },
            isLoading: false,
          }),
        },
      },
    },
    ratelimit: {
      namespace: { list: { useQuery: () => ({ data: [], isLoading: false }) } },
    },
  },
}));

afterEach(cleanup);

const handlers = () => ({
  onChange: vi.fn(),
  onRemove: vi.fn(),
  onCollapsedChange: vi.fn(),
});

describe("PolicyCard collapsed", () => {
  it("summarises the scope and the first three grants", () => {
    const rows = ["identity", "role", "permission", "vault_key"];
    const selection = rows.reduce((acc, row) => setRowActions(acc, row, ["read"]), {});
    render(
      <PolicyCard
        policy={{ ...newPolicy(), selection }}
        collapsed
        showError={false}
        {...handlers()}
      />,
    );

    expect(screen.getByText("All resources")).toBeDefined();
    expect(
      screen.getByText(
        /End-user identities Read, Role definitions Read, Permission definitions Read \+1 more…/,
      ),
    ).toBeDefined();
  });

  it("says a policy without grants is incomplete", () => {
    render(<PolicyCard policy={newPolicy()} collapsed showError={false} {...handlers()} />);

    expect(screen.getByText("At least one permission required")).toBeDefined();
  });

  it("names the picked keyspace instead of its id", () => {
    render(
      <PolicyCard
        policy={{
          ...newPolicy("keyspaces"),
          instances: ["ks_1"],
          selection: setRowActions({}, "key", ["read"]),
        }}
        collapsed
        showError={false}
        {...handlers()}
      />,
    );

    expect(screen.getByText("payments")).toBeDefined();
    expect(screen.getByText("API keys Read")).toBeDefined();
  });

  it("expands on click", () => {
    const props = handlers();
    render(<PolicyCard policy={newPolicy()} collapsed showError={false} {...props} />);
    fireEvent.click(screen.getByText("All resources"));

    expect(props.onCollapsedChange).toHaveBeenCalledWith(false);
  });
});

describe("PolicyCard expanded", () => {
  it("resets the selection and the instances when the scope changes", () => {
    const props = handlers();
    render(
      <PolicyCard
        policy={{
          ...newPolicy("keyspaces"),
          instances: ["ks_1"],
          selection: setRowActions({}, "key", ["read"]),
        }}
        collapsed={false}
        showError={false}
        {...props}
      />,
    );
    fireEvent.change(screen.getByLabelText("Resource type"), {
      target: { value: "ratelimit-namespaces" },
    });

    expect(props.onChange).toHaveBeenCalledWith({
      scope: "ratelimit-namespaces",
      instances: [ALL_INSTANCES],
      selection: {},
    });
  });

  it("offers the nine resource scopes as plain nouns", () => {
    render(<PolicyCard policy={newPolicy()} collapsed={false} showError={false} {...handlers()} />);
    const options = Array.from(screen.getByLabelText("Resource type").children).map(
      (option) => option.textContent,
    );

    expect(options).toEqual([
      "Workspace",
      "Projects",
      "Apps",
      "Environments",
      "Keyspaces",
      "Ratelimit namespaces",
      "Identities",
      "RBAC",
      "Vault",
    ]);
  });

  it("names the picked environment by its app and its own name", () => {
    render(
      <PolicyCard
        policy={{ ...newPolicy("environments"), instances: ["env_1"] }}
        collapsed={false}
        showError={false}
        {...handlers()}
      />,
    );

    expect(screen.getByLabelText("Select environments").textContent).toContain("site production");
  });

  it("hides the instance picker on the workspace scope", () => {
    render(<PolicyCard policy={newPolicy()} collapsed={false} showError={false} {...handlers()} />);

    expect(screen.queryByLabelText(/^Select /)).toBeNull();
  });

  it("hides the instance picker on the container-less scopes", () => {
    for (const scope of ["identities", "rbac", "vault"] as const) {
      cleanup();
      render(
        <PolicyCard
          policy={newPolicy(scope)}
          collapsed={false}
          showError={false}
          {...handlers()}
        />,
      );
      expect(screen.queryByLabelText(/^Select /)).toBeNull();
    }
  });

  it("labels the scope row", () => {
    render(<PolicyCard policy={newPolicy()} collapsed={false} showError={false} {...handlers()} />);

    expect(screen.getByText("Scope")).toBeDefined();
  });

  it("shows the instance picker on an instance scope", () => {
    render(
      <PolicyCard
        policy={newPolicy("keyspaces")}
        collapsed={false}
        showError={false}
        {...handlers()}
      />,
    );

    expect(screen.getByLabelText("Select keyspaces").textContent).toContain("All keyspaces");
  });

  it("only shows the error once the form was submitted", () => {
    const props = handlers();
    const { rerender } = render(
      <PolicyCard policy={newPolicy()} collapsed={false} showError={false} {...props} />,
    );
    expect(screen.queryByText("At least one permission required")).toBeNull();

    rerender(<PolicyCard policy={newPolicy()} collapsed={false} showError {...props} />);
    expect(screen.getByText("At least one permission required")).toBeDefined();
  });

  it("removes and collapses through its own controls", () => {
    const props = handlers();
    render(<PolicyCard policy={newPolicy()} collapsed={false} showError={false} {...props} />);

    fireEvent.click(screen.getByLabelText("Delete policy"));
    expect(props.onRemove).toHaveBeenCalled();

    fireEvent.click(screen.getByLabelText("Close policy"));
    expect(props.onCollapsedChange).toHaveBeenCalledWith(true);
  });
});
