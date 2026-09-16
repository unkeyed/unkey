import { fireEvent, render, screen } from "@testing-library/react";
import type { ChangeEvent, ReactNode } from "react";
import { useState } from "react";
import { describe, expect, it, vi } from "vitest";
import { PermissionField } from "./permissions-field";

const mocks = vi.hoisted(() => ({
  searchedPermission: {
    id: "perm_billing",
    name: "Manage Billing",
    slug: "billing.manage",
    description: "Manage billing settings",
    roles: [],
  },
}));

vi.mock("@/components/roles-table/hooks/use-role-limits", () => ({
  useRoleLimits: () => ({
    calculateLimits: () => ({ hasPermWarning: false, totalPerms: 0 }),
  }),
}));

vi.mock("./hooks/use-fetch-permissions", () => ({
  useFetchPermissions: () => ({
    permissions: [],
    isFetchingNextPage: false,
    hasNextPage: false,
    loadMore: vi.fn(),
    isLoading: false,
  }),
}));

vi.mock("./hooks/use-search-permissions", () => ({
  useSearchPermissions: (query: string) => ({
    searchResults: query ? [mocks.searchedPermission] : [],
    isSearching: false,
  }),
}));

vi.mock("@/components/ui/form-combobox", () => ({
  FormCombobox: ({
    onChange,
    onSelect,
  }: {
    onChange: (event: ChangeEvent<HTMLInputElement>) => void;
    onSelect: (value: string) => void;
  }) => (
    <div>
      <input aria-label="Search permissions" onChange={onChange} />
      <button type="button" onClick={() => onSelect(mocks.searchedPermission.id)}>
        Select permission
      </button>
    </div>
  ),
}));

vi.mock("@/components/selected-item-list", () => ({
  SelectedItemsList: ({
    items,
    renderPrimaryText,
    renderSecondaryText,
  }: {
    items: Array<{ id: string; name: string; slug: string | null }>;
    renderPrimaryText: (item: { id: string; name: string; slug: string | null }) => ReactNode;
    renderSecondaryText: (item: { id: string; name: string; slug: string | null }) => ReactNode;
  }) => (
    <div>
      {items.map((item) => (
        <div key={item.id}>
          <span>{renderPrimaryText(item)}</span>
          <span>{renderSecondaryText(item)}</span>
        </div>
      ))}
    </div>
  ),
}));

vi.mock("../warning-callout", () => ({ RoleWarningCallout: () => null }));

describe("PermissionField", () => {
  it("keeps permission details after clearing a search", () => {
    const TestField = () => {
      const [value, setValue] = useState<string[]>([]);

      return <PermissionField value={value} onChange={setValue} assignedPermsDetails={[]} />;
    };

    render(<TestField />);

    fireEvent.change(screen.getByRole("textbox", { name: "Search permissions" }), {
      target: { value: "billing" },
    });
    fireEvent.click(screen.getByRole("button", { name: "Select permission" }));

    expect(screen.getByText("Manage Billing")).not.toBeNull();
    expect(screen.getByText("billing.manage")).not.toBeNull();
    expect(screen.queryByText("Unnamed Permission")).toBeNull();
  });
});
