import { SelectedItemsList } from "@/components/selected-item-list";
import { FormCombobox } from "@/components/ui/form-combobox";
import type { KeyRole } from "@/lib/trpc/routers/key/rbac/connected-roles-and-perms";
import { Tag } from "@unkey/icons";
import { useMemo, useState } from "react";
import { createRoleOptions } from "./create-key-options";
import { useFetchKeysRoles } from "./hooks/use-fetch-keys-roles";
import { useSearchKeysRoles } from "./hooks/use-search-keys-roles";

type RoleFieldProps = {
  value: string[];
  onChange: (ids: string[]) => void;
  error?: string;
  disabled?: boolean;
  keyId?: string;
  assignedRoleDetails: KeyRole[];
};

export const RoleField = ({
  value,
  onChange,
  error,
  disabled = false,
  keyId,
  assignedRoleDetails,
}: RoleFieldProps) => {
  const [searchValue, setSearchValue] = useState("");
  const trimmedSearchVal = searchValue.trim();
  const { roles, isFetchingNextPage, hasNextPage, loadMore, isLoading } =
    useFetchKeysRoles();
  const { searchResults, isSearching } = useSearchKeysRoles(searchValue);

  // Combine loaded roles with search results, prioritizing search when available
  const allRoles = useMemo(() => {
    if (trimmedSearchVal && searchResults.length > 0) {
      return searchResults;
    }
    if (trimmedSearchVal && searchResults.length === 0 && !isSearching) {
      const searchTerm = trimmedSearchVal.toLowerCase();
      return roles.filter(
        (role) =>
          role.id.toLowerCase().includes(searchTerm) ||
          role.name?.toLowerCase().includes(searchTerm) ||
          role.description?.toLowerCase().includes(searchTerm)
      );
    }
    return roles;
  }, [roles, searchResults, trimmedSearchVal, isSearching]);

  const showLoadMore = !trimmedSearchVal && hasNextPage;

  const baseOptions = createRoleOptions({
    roles: allRoles,
    hasNextPage: showLoadMore,
    isFetchingNextPage,
    keyId,
    previouslySelectedRoleIds: assignedRoleDetails.map((r) => r.id),
    loadMore,
  });

  const selectableOptions = useMemo(() => {
    return baseOptions.filter((option) => {
      if (option.value === "__load_more__") {
        return true;
      }

      // Don't show already selected roles (based on current form state)
      if (value.includes(option.value)) {
        return false;
      }

      return true;
    });
  }, [baseOptions, value]);

  const selectedRoles = useMemo(() => {
    return value
      .map((roleId) => {
        // Check assignedRoleDetails first (for pre-loaded edit data)
        const preLoadedRole = assignedRoleDetails.find((r) => r.id === roleId);
        if (preLoadedRole) {
          return {
            id: preLoadedRole.id,
            name: preLoadedRole.name,
            description: preLoadedRole.description,
          };
        }

        // Check loaded roles (for newly added roles)
        const loadedRole = allRoles.find((r) => r.id === roleId);
        if (loadedRole) {
          return {
            id: loadedRole.id,
            name: loadedRole.name,
            description: loadedRole.description,
          };
        }

        // Fallback to ID-only display
        return {
          id: roleId,
          name: null,
          description: null,
        };
      })
      .filter((role): role is NonNullable<typeof role> => role !== undefined);
  }, [value, allRoles, assignedRoleDetails]);

  const handleRemoveRole = (roleId: string) => {
    onChange(value.filter((id) => id !== roleId));
  };

  const handleAddRole = (roleId: string) => {
    if (!value.includes(roleId)) {
      onChange([...value, roleId]);
    }
    setSearchValue("");
  };

  const isComboboxLoading =
    isLoading || (isSearching && trimmedSearchVal.length > 0);
  return (
    <div className="space-y-3">
      <FormCombobox
        label="Assign roles"
        description="Select roles from your workspace."
        options={selectableOptions}
        value=""
        onChange={(e) => setSearchValue(e.currentTarget.value)}
        onSelect={handleAddRole}
        placeholder={
          <div className="flex w-full text-grayA-8 text-[13px] gap-1.5 items-center py-2">
            Select roles
          </div>
        }
        searchPlaceholder="Search roles by name or description..."
        emptyMessage={
          isComboboxLoading ? (
            <div className="px-3 py-3 text-gray-10 text-[13px] flex items-center gap-2">
              <div className="animate-spin h-3 w-3 border border-gray-6 border-t-gray-11 rounded-full" />
              {isSearching ? "Searching..." : "Loading roles..."}
            </div>
          ) : (
            <div className="px-3 py-3 text-gray-10 text-[13px]">
              No roles found
            </div>
          )
        }
        variant="default"
        error={error}
        disabled={disabled || isLoading}
        loading={isComboboxLoading}
        title={
          isComboboxLoading
            ? isSearching && trimmedSearchVal
              ? "Searching for roles..."
              : "Loading available roles..."
            : undefined
        }
      />

      <SelectedItemsList
        items={selectedRoles.map((r) => ({
          name: r.name ?? "",
          description: r.description ?? "",
          id: r.id,
        }))}
        disabled={disabled}
        onRemoveItem={handleRemoveRole}
        renderIcon={() => (
          <Tag iconSize="sm-regular" className="text-grayA-11" />
        )}
        enableTransitions
        renderPrimaryText={(role) => role.name || "Unnamed Role"}
        renderSecondaryText={(role) =>
          ("description" in role && role.description) || role.id
        }
      />
    </div>
  );
};
