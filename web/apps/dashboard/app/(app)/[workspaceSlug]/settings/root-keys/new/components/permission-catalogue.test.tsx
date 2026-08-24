import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ButtonHTMLAttributes, InputHTMLAttributes, ReactNode } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { PermissionSelection } from "../lib/catalogue.types";
import { workspaceCatalogue } from "../lib/catalogue.workspace";
import { setRowActions } from "../lib/policy";
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

const renderCatalogue = (selection: PermissionSelection = {}) => {
  const onChange = vi.fn();
  render(
    <PermissionCatalogue catalogue={workspaceCatalogue} value={selection} onChange={onChange} />,
  );
  return onChange;
};

const search = (value: string) => {
  fireEvent.change(screen.getByPlaceholderText("Search permissions…"), { target: { value } });
};

describe("PermissionCatalogue", () => {
  it("lists every group with a selected-of-total counter", () => {
    renderCatalogue(setRowActions({}, "role", ["read", "write"]));

    expect(screen.getByText("Identities")).toBeDefined();
    expect(screen.getByText("RBAC")).toBeDefined();
    expect(screen.getByText("Vault")).toBeDefined();
    expect(screen.getByText("2/6")).toBeDefined();
    expect(screen.getAllByText("0/3")).toHaveLength(2);
  });

  it("starts with every group expanded", () => {
    renderCatalogue();

    expect(screen.getByText("Role definitions")).toBeDefined();
    expect(screen.getByText("Encryption keys")).toBeDefined();
  });

  it("collapses a group on its trigger", () => {
    renderCatalogue();
    fireEvent.click(screen.getByText("RBAC"));

    expect(screen.queryByText("Role definitions")).toBeNull();
  });

  it("reports selected rows the filter hides", () => {
    renderCatalogue(setRowActions({}, "role", ["read"]));
    search("vault");

    expect(screen.getByText("1 selected row hidden by this filter")).toBeDefined();
    expect(screen.queryByText("Role definitions")).toBeNull();
  });

  it("stays quiet when the filter hides nothing that is selected", () => {
    renderCatalogue(setRowActions({}, "vault_key", ["read"]));
    search("encryption");

    expect(screen.queryByText(/hidden by this filter/)).toBeNull();
    expect(screen.getByText("Encryption keys")).toBeDefined();
  });

  it("says so when nothing matches the filter", () => {
    renderCatalogue();
    search("nothing here");

    expect(screen.getByText("No permissions match “nothing here”.")).toBeDefined();
  });

  it("reports one row toggle as a selection change", () => {
    const onChange = renderCatalogue();
    search("role");
    fireEvent.click(screen.getByLabelText("Read"));

    expect(onChange).toHaveBeenCalledWith({ role: ["read"] });
  });
});
