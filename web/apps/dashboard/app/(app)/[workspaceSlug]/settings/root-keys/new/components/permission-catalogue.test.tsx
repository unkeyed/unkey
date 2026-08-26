import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { PermissionSelection } from "../lib/catalogue.types";
import { workspaceCatalogue } from "../lib/catalogue.workspace";
import { ALL_INSTANCES, setRowActions } from "../lib/policy";
import { PermissionCatalogue } from "./permission-catalogue";

vi.mock("@unkey/icons", () => ({ ChevronRight: () => null, ChevronDown: () => null }));

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
}));

afterEach(cleanup);

const renderCatalogue = (selection: PermissionSelection = {}, debug = false) => {
  const onChange = vi.fn();
  render(
    <PermissionCatalogue
      catalogue={workspaceCatalogue}
      instances={[ALL_INSTANCES]}
      debug={debug}
      value={selection}
      onChange={onChange}
    />,
  );
  return onChange;
};

const search = (value: string) => {
  fireEvent.change(screen.getByPlaceholderText("Search permissions…"), { target: { value } });
};

describe("PermissionCatalogue", () => {
  it("gives every scope family its own group", () => {
    renderCatalogue();

    expect(
      screen
        .getAllByRole("button")
        .map((trigger) => trigger.textContent ?? "")
        .filter((label) => /\d+\/\d+$/.test(label))
        .map((label) => label.replace(/\d+\/\d+$/, "")),
    ).toEqual([
      "Projects",
      "Apps",
      "Environments",
      "Deployments",
      "Gateway",
      "Key management",
      "Rate limiting",
      "Identity and RBAC",
      "Connections",
    ]);
  });

  it("counts the selected actions of a group against its total", () => {
    renderCatalogue(setRowActions({}, "role", ["read_role", "write_role"]));

    expect(screen.getByText("2/9")).toBeDefined();
    expect(screen.getAllByText("0/9")).toHaveLength(2);
    expect(screen.getByText("0/8")).toBeDefined();
    expect(screen.getByText("0/6")).toBeDefined();
    expect(screen.getByText("0/4")).toBeDefined();
    expect(screen.getAllByText("0/3")).toHaveLength(3);
  });

  it("starts with every group expanded", () => {
    renderCatalogue();

    expect(screen.getByText("Roles")).toBeDefined();
    expect(screen.getByText("GitHub apps")).toBeDefined();
  });

  it("collapses a group on its trigger", () => {
    renderCatalogue();
    fireEvent.click(screen.getByText("Identity and RBAC"));

    expect(screen.queryByText("Roles")).toBeNull();
  });

  it("reports selected rows the filter hides", () => {
    renderCatalogue(setRowActions({}, "role", ["read_role"]));
    search("github");

    expect(screen.getByText("1 selected row hidden by this filter")).toBeDefined();
    expect(screen.queryByText("Roles")).toBeNull();
  });

  it("stays quiet when the filter hides nothing that is selected", () => {
    renderCatalogue(setRowActions({}, "github_app", ["read_github_app"]));
    search("github");

    expect(screen.queryByText(/hidden by this filter/)).toBeNull();
    expect(screen.getByText("GitHub apps")).toBeDefined();
  });

  it("says so when nothing matches the filter", () => {
    renderCatalogue();
    search("nothing here");

    expect(screen.getByText("No permissions match “nothing here”.")).toBeDefined();
  });

  it("keeps the grant mapping out of the way until debug is on", () => {
    renderCatalogue();
    expect(screen.queryByText(/rbac\/roles/)).toBeNull();

    cleanup();
    renderCatalogue({}, true);
    expect(screen.getByText("projects/*/rbac/roles/*")).toBeDefined();
    expect(screen.getByText("projects/*/keyspaces/*/keys/*")).toBeDefined();
  });

  it("reports one row toggle as a selection change", () => {
    const onChange = renderCatalogue();
    search("role");
    fireEvent.click(screen.getByLabelText("Read"));

    expect(onChange).toHaveBeenCalledWith({ role: ["read_role"] });
  });
});
