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
            data: { ks_1: { id: "ks_1", projectId: "proj_1", api: { name: "payments" } } },
            isLoading: false,
          }),
        },
      },
    },
    ratelimit: {
      namespace: {
        list: {
          useQuery: () => ({
            data: [{ id: "rlns_1", projectId: "proj_1", name: "api" }],
            isLoading: false,
          }),
        },
      },
    },
  },
}));

afterEach(cleanup);

const defaults = () => ({
  debug: false,
  onChange: vi.fn(),
  onRemove: vi.fn(),
  onCollapsedChange: vi.fn(),
});

describe("PolicyCard collapsed", () => {
  it("summarises the scope and the first three grants", () => {
    const selection = setRowActions(
      setRowActions(
        setRowActions(setRowActions({}, "github_app", ["read_github_app"]), "identity", [
          "read_identity",
        ]),
        "role",
        ["read_role"],
      ),
      "permission",
      ["read_permission"],
    );
    render(
      <PolicyCard
        policy={{ ...newPolicy(), selection }}
        collapsed
        showError={false}
        {...defaults()}
      />,
    );

    expect(screen.getByText("All resources")).toBeDefined();
    expect(
      screen.getByText(/Identities Read, Roles Read, Permissions Read \+1 more…/),
    ).toBeDefined();
  });

  it("says a policy without grants is incomplete", () => {
    render(<PolicyCard policy={newPolicy()} collapsed showError={false} {...defaults()} />);

    expect(screen.getByText("At least one permission required")).toBeDefined();
  });

  it("names the picked keyspace instead of its id", () => {
    render(
      <PolicyCard
        policy={{
          ...newPolicy("keyspaces"),
          instances: ["projects/proj_1/keyspaces/ks_1"],
          selection: setRowActions({}, "key", ["read_key"]),
        }}
        collapsed
        showError={false}
        {...defaults()}
      />,
    );

    expect(screen.getByText("payments")).toBeDefined();
    expect(screen.getByText("Keys Read")).toBeDefined();
  });

  it("expands on click", () => {
    const props = defaults();
    render(<PolicyCard policy={newPolicy()} collapsed showError={false} {...props} />);
    fireEvent.click(screen.getByText("All resources"));

    expect(props.onCollapsedChange).toHaveBeenCalledWith(false);
  });
});

describe("PolicyCard expanded", () => {
  it("resets the selection and the instances when the scope changes", () => {
    const props = defaults();
    render(
      <PolicyCard
        policy={{
          ...newPolicy("keyspaces"),
          instances: ["projects/proj_1/keyspaces/ks_1"],
          selection: setRowActions({}, "key", ["read_key"]),
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

  it("offers the eight resource scopes as narrowing choices", () => {
    render(<PolicyCard policy={newPolicy()} collapsed={false} showError={false} {...defaults()} />);
    const options = Array.from(screen.getByLabelText("Resource type").children).map(
      (option) => option.textContent,
    );

    expect(options).toEqual([
      "Entire workspace",
      "Specific projects",
      "Specific apps",
      "Specific environments",
      "Specific keyspaces",
      "Specific rate limit namespaces",
      "All identities",
      "All roles and permissions",
    ]);
  });

  it("names the picked environment by its app and its own name", () => {
    render(
      <PolicyCard
        policy={{
          ...newPolicy("environments"),
          instances: ["projects/proj_1/apps/app_1/environments/env_1"],
        }}
        collapsed={false}
        showError={false}
        {...defaults()}
      />,
    );

    expect(screen.getByLabelText("Select environments").textContent).toContain("site production");
  });

  it("hides the instance picker on the workspace scope", () => {
    render(<PolicyCard policy={newPolicy()} collapsed={false} showError={false} {...defaults()} />);

    expect(screen.queryByLabelText(/^Select /)).toBeNull();
  });

  it("hides the instance picker on the container-less scopes", () => {
    for (const scope of ["identities", "rbac"] as const) {
      cleanup();
      render(
        <PolicyCard
          policy={newPolicy(scope)}
          collapsed={false}
          showError={false}
          {...defaults()}
        />,
      );
      expect(screen.queryByLabelText(/^Select /)).toBeNull();
    }
  });

  it("labels the scope row", () => {
    render(<PolicyCard policy={newPolicy()} collapsed={false} showError={false} {...defaults()} />);

    expect(screen.getByText("Scope")).toBeDefined();
  });

  it("shows the instance picker on an instance scope", () => {
    render(
      <PolicyCard
        policy={newPolicy("keyspaces")}
        collapsed={false}
        showError={false}
        {...defaults()}
      />,
    );

    expect(screen.getByLabelText("Select keyspaces").textContent).toContain("All keyspaces");
  });

  it("only shows the error once the form was submitted", () => {
    const props = defaults();
    const { rerender } = render(
      <PolicyCard policy={newPolicy()} collapsed={false} showError={false} {...props} />,
    );
    expect(screen.queryByText("At least one permission required")).toBeNull();

    rerender(<PolicyCard policy={newPolicy()} collapsed={false} showError {...props} />);
    expect(screen.getByText("At least one permission required")).toBeDefined();
  });

  it("removes and collapses through its own controls", () => {
    const props = defaults();
    render(<PolicyCard policy={newPolicy()} collapsed={false} showError={false} {...props} />);

    fireEvent.click(screen.getByLabelText("Delete policy"));
    expect(props.onRemove).toHaveBeenCalled();

    fireEvent.click(screen.getByLabelText("Close policy"));
    expect(props.onCollapsedChange).toHaveBeenCalledWith(true);
  });
});
