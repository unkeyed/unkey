import { cleanup, fireEvent, render, screen } from "@testing-library/react";
import type { ButtonHTMLAttributes, InputHTMLAttributes } from "react";
import { afterEach, describe, expect, it, vi } from "vitest";
import type { PermissionSelection } from "./lib/catalogue.types";
import { workspaceCatalogue } from "./lib/catalogue.workspace";
import { setRowActions } from "./lib/policy";
import { PermissionCatalogue } from "./permission-catalogue";

// @unkey/ui declares react as a dependency rather than a peer, so under vitest
// its primitives run against a second React copy and every hook call throws.
// Until that is fixed the hook-using primitives have to be stubbed; everything
// else here — icons, the collapsible, the catalogue itself — renders for real.
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
      "Keyspaces",
      "Ratelimit namespaces",
      "Identities",
      "RBAC",
    ]);
  });

  it("counts the selected actions of a group against its total", () => {
    renderCatalogue(setRowActions({}, "role", ["read", "write"]));

    expect(screen.getByText("2/6")).toBeDefined();
    expect(screen.getByText("0/12")).toBeDefined();
    expect(screen.getAllByText("0/3")).toHaveLength(3);
  });

  it("counts the decrypt action of the api keys row in the keyspaces total", () => {
    renderCatalogue();

    expect(screen.getByText("0/7")).toBeDefined();
  });

  it("starts with every group expanded", () => {
    renderCatalogue();

    expect(screen.getByText("Role definitions")).toBeDefined();
    expect(screen.getByText("API keys")).toBeDefined();
  });

  it("collapses a group on its trigger", () => {
    renderCatalogue();
    fireEvent.click(screen.getByText("RBAC"));

    expect(screen.queryByText("Role definitions")).toBeNull();
  });

  it("reports selected rows the filter hides", () => {
    renderCatalogue(setRowActions({}, "role", ["read"]));
    search("api keys");

    expect(screen.getByText("1 selected row hidden by this filter")).toBeDefined();
    expect(screen.queryByText("Role definitions")).toBeNull();
  });

  it("stays quiet when the filter hides nothing that is selected", () => {
    renderCatalogue(setRowActions({}, "key", ["read"]));
    search("api keys");

    expect(screen.queryByText(/hidden by this filter/)).toBeNull();
    expect(screen.getByText("API keys")).toBeDefined();
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

  it("offers the decrypt checkbox on the api keys row alone", () => {
    renderCatalogue();
    search("api keys");
    expect(screen.getByLabelText("Decrypt")).toBeDefined();

    search("role");
    expect(screen.queryByLabelText("Decrypt")).toBeNull();
  });

  it("reports a decrypt tick as a selection change", () => {
    const onChange = renderCatalogue();
    search("api keys");
    fireEvent.click(screen.getByLabelText("Decrypt"));

    expect(onChange).toHaveBeenCalledWith({ key: ["decrypt"] });
  });
});
