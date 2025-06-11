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
  const { roles, isFetchingNextPage, hasNextPage, loadMore } = useFetchKeysRoles();
  const { searchResults, isSearching } = useSearchKeysRoles(searchValue);

  // Combine loaded roles with search results, prioritizing search when available
  const allRoles = useMemo(() => {
    if (searchValue.trim() && searchResults.length > 0) {
      // When searching, use search results
      return searchResults;
    }
    if (searchValue.trim() && searchResults.length === 0 && !isSearching) {
      // No search results found, filter from loaded roles as fallback
      const searchTerm = searchValue.toLowerCase().trim();
      return roles.filter(
        (role) =>
          role.id.toLowerCase().includes(searchTerm) ||
          role.name?.toLowerCase().includes(searchTerm) ||
          role.description?.toLowerCase().includes(searchTerm),
      );
    }
    // No search query, use all loaded roles
    return roles;
  }, [roles, searchResults, searchValue, isSearching]);

  // Don't show load more when actively searching
  const showLoadMore = !searchValue.trim() && hasNextPage;

  const baseOptions = createRoleOptions({
    roles: allRoles,
    hasNextPage: showLoadMore,
    isFetchingNextPage,
    keyId,
    loadMore,
  });

  const selectableOptions = useMemo(() => {
    return baseOptions.filter((option) => {
      // Always allow the load more option
      if (option.value === "__load_more__") {
        return true;
      }
      // Don't show already selected roles
      if (value.includes(option.value)) {
        return false;
      }
      // Find the role and check if it's already assigned to this key
      const role = allRoles.find((r) => r.id === option.value);
      if (!role) {
        return true;
      }
      // Filter out roles that already have this key assigned (if keyId provided)
      if (keyId) {
        return !role.keys.some((key) => key.id === keyId);
      }
      return true;
    });
  }, [baseOptions, allRoles, keyId, value]);

  const selectedRoles = useMemo(() => {
    return value
      .map((roleId) => {
        // First: check assignedRoleDetails (for pre-loaded edit data)
        const preLoadedRole = assignedRoleDetails.find((r) => r.id === roleId);
        if (preLoadedRole) {
          return {
            id: preLoadedRole.id,
            name: preLoadedRole.name,
            description: preLoadedRole.description,
          };
        }
        // Second: check loaded roles (for newly added roles)
        const loadedRole = allRoles.find((r) => r.id === roleId);
        if (loadedRole) {
          return {
            id: loadedRole.id,
            name: loadedRole.name,
            description: loadedRole.description,
          };
        }
        // Third: fallback to ID-only display (ensures role is always removable)
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

  return (
    <div className="space-y-3">
      <FormCombobox
        label="Assign roles"
        description="Select roles from your workspace."
        options={selectableOptions}
        value=""
        onChange={(e) => setSearchValue(e.currentTarget.value)}
        onSelect={(val) => {
          if (val === "__load_more__") {
            return;
          }
          // Add the selected role to the array
          if (!value.includes(val)) {
            onChange([...value, val]);
          }
          // Clear search after selection
          setSearchValue("");
        }}
        placeholder={
          <div className="flex w-full text-grayA-8 text-[13px] gap-1.5 items-center py-2">
            Select roles
          </div>
        }
        searchPlaceholder="Search roles by name or description..."
        emptyMessage={
          isSearching ? (
            <div className="px-3 py-3 text-gray-10 text-[13px]">Searching...</div>
          ) : (
            <div className="px-3 py-3 text-gray-10 text-[13px]">No roles found</div>
          )
        }
        variant="default"
        error={error}
        disabled={disabled}
      />

      <SelectedItemsList
        items={selectedRoles}
        disabled={disabled}
        onRemoveItem={handleRemoveRole}
        renderIcon={() => (
          <div className="border rounded flex items-center justify-center border-grayA-4 bg-grayA-3 flex-shrink-0 size-5">
            <Tag size="sm-regular" className="text-grayA-11" />
          </div>
        )}
        renderPrimaryText={(role) => role.name || "Unnamed Role"}
        renderSecondaryText={(role) => ("description" in role && role.description) || role.id}
      />
    </div>
  );
};
