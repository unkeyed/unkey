import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import { ALL_INSTANCES } from "./lib/catalogue.types";
import { newPolicy, setRowActions } from "./lib/policy";
import { PolicyEditor, PolicySummaryRow } from "./policy-card";

// @unkey/ui declares react as a dependency rather than a peer, so under vitest
// its primitives run against a second React copy and every hook call throws.
// Only the hook-using primitives are stubbed; icons and the instance picker
// render for real.
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

const summaryProps = () => ({ onExpand: vi.fn(), onRemove: vi.fn() });

const editorProps = () => ({ onChange: vi.fn(), onCollapse: vi.fn(), onRemove: vi.fn() });

describe("PolicySummaryRow", () => {
  it("summarises the scope and the first three grants", () => {
    const rows = ["key", "identity", "role", "permission"];
    const selection = rows.reduce((acc, row) => setRowActions(acc, row, ["read"]), {});
    render(<PolicySummaryRow policy={{ ...newPolicy(), selection }} {...summaryProps()} />);

    expect(screen.getByText("All resources")).toBeDefined();
    expect(
      screen.getByText(/API keys Read, End-user identities Read, Role definitions Read \+1 more…/),
    ).toBeDefined();
  });

  it("says a policy without grants is incomplete", () => {
    render(<PolicySummaryRow policy={newPolicy()} {...summaryProps()} />);

    expect(screen.getByText("At least one permission required")).toBeDefined();
  });

  it("names the picked keyspace instead of its id", () => {
    render(
      <PolicySummaryRow
        policy={{
          ...newPolicy("keyspaces"),
          instances: ["ks_1"],
          selection: setRowActions({}, "key", ["read"]),
        }}
        {...summaryProps()}
      />,
    );

    expect(screen.getByText("payments")).toBeDefined();
    expect(screen.getByText("API keys Read")).toBeDefined();
  });

  it("expands on click", () => {
    const props = summaryProps();
    render(<PolicySummaryRow policy={newPolicy()} {...props} />);
    fireEvent.click(screen.getByText("All resources"));

    expect(props.onExpand).toHaveBeenCalled();
  });

  it("removes through its own control", () => {
    const props = summaryProps();
    render(<PolicySummaryRow policy={newPolicy()} {...props} />);
    fireEvent.click(screen.getByLabelText("Delete policy"));

    expect(props.onRemove).toHaveBeenCalled();
  });
});

describe("PolicyEditor", () => {
  it("resets the selection and the instances when the scope changes", () => {
    const props = editorProps();
    render(
      <PolicyEditor
        policy={{
          ...newPolicy("keyspaces"),
          instances: ["ks_1"],
          selection: setRowActions({}, "key", ["read"]),
        }}
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

  it("offers the eight resource scopes as plain nouns", () => {
    render(<PolicyEditor policy={newPolicy()} {...editorProps()} />);
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
    ]);
  });

  it("names the picked environment by its app and its own name", () => {
    render(
      <PolicyEditor
        policy={{ ...newPolicy("environments"), instances: ["env_1"] }}
        {...editorProps()}
      />,
    );

    expect(screen.getByLabelText("Select environments").textContent).toContain("site production");
  });

  it("hides the instance picker on the workspace scope", () => {
    render(<PolicyEditor policy={newPolicy()} {...editorProps()} />);

    expect(screen.queryByLabelText(/^Select /)).toBeNull();
  });

  it("hides the instance picker on the container-less scopes", () => {
    for (const scope of ["identities", "rbac"] as const) {
      cleanup();
      render(<PolicyEditor policy={newPolicy(scope)} {...editorProps()} />);
      expect(screen.queryByLabelText(/^Select /)).toBeNull();
    }
  });

  it("labels the scope row", () => {
    render(<PolicyEditor policy={newPolicy()} {...editorProps()} />);

    expect(screen.getByText("Scope")).toBeDefined();
  });

  it("shows the instance picker on an instance scope", () => {
    render(<PolicyEditor policy={newPolicy("keyspaces")} {...editorProps()} />);

    expect(screen.getByLabelText("Select keyspaces").textContent).toContain("All keyspaces");
  });

  it("shows only the error the form hands it", () => {
    const props = editorProps();
    const { rerender } = render(<PolicyEditor policy={newPolicy()} {...props} />);
    expect(screen.queryByText("At least one permission required")).toBeNull();

    rerender(
      <PolicyEditor policy={newPolicy()} error="At least one permission required" {...props} />,
    );
    expect(screen.getByText("At least one permission required")).toBeDefined();
  });

  it("removes and collapses through its own controls", () => {
    const props = editorProps();
    render(<PolicyEditor policy={newPolicy()} {...props} />);

    fireEvent.click(screen.getByLabelText("Delete policy"));
    expect(props.onRemove).toHaveBeenCalled();

    fireEvent.click(screen.getByLabelText("Close policy"));
    expect(props.onCollapse).toHaveBeenCalled();
  });
});
