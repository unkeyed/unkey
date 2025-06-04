import { FormCombobox } from "@/components/ui/form-combobox";
import type { RolePermission } from "@/lib/trpc/routers/authorization/roles/connected-keys-and-perms";
import { Key2, XMark } from "@unkey/icons";
import { useMemo, useState } from "react";
import { createPermissionOptions } from "./create-permission-options";
import { useFetchPermissions } from "./hooks/use-fetch-permissions";
import { useSearchPermissions } from "./hooks/use-search-permissions";

type PermissionFieldProps = {
  value: string[];
  onChange: (ids: string[]) => void;
  error?: string;
  disabled?: boolean;
  roleId?: string;
  assignedPermsDetails: RolePermission[];
};

export const PermissionField = ({
  value,
  onChange,
  error,
  disabled = false,
  roleId,
  assignedPermsDetails = [],
}: PermissionFieldProps) => {
  const [searchValue, setSearchValue] = useState("");
  const { permissions, isFetchingNextPage, hasNextPage, loadMore } = useFetchPermissions();
  const { searchResults, isSearching } = useSearchPermissions(searchValue);

  // Combine loaded permissions with search results, prioritizing search when available
  const allPermissions = useMemo(() => {
    if (searchValue.trim() && searchResults.length > 0) {
      // When searching, use search results
      return searchResults;
    }
    if (searchValue.trim() && searchResults.length === 0 && !isSearching) {
      // No search results found, filter from loaded permissions as fallback
      const searchTerm = searchValue.toLowerCase().trim();
      return permissions.filter(
        (permission) =>
          permission.id.toLowerCase().includes(searchTerm) ||
          permission.name.toLowerCase().includes(searchTerm) ||
          permission.slug.toLowerCase().includes(searchTerm) ||
          permission.description?.toLowerCase().includes(searchTerm),
      );
    }
    // No search query, use all loaded permissions
    return permissions;
  }, [permissions, searchResults, searchValue, isSearching]);

  // Don't show load more when actively searching
  const showLoadMore = !searchValue.trim() && hasNextPage;

  const baseOptions = createPermissionOptions({
    permissions: allPermissions,
    hasNextPage: showLoadMore,
    isFetchingNextPage,
    roleId,
    loadMore,
  });

  const selectableOptions = useMemo(() => {
    return baseOptions.filter((option) => {
      // Always allow the load more option
      if (option.value === "__load_more__") {
        return true;
      }

      // Don't show already selected permissions
      if (value.includes(option.value)) {
        return false;
      }

      // Find the permission and check if it's already assigned to this role
      const permission = allPermissions.find((p) => p.id === option.value);
      if (!permission) {
        return true;
      }

      // Filter out permissions that already have this role assigned (if roleId provided)
      if (roleId) {
        return !permission.roles.some((role) => role.id === roleId);
      }

      return true;
    });
  }, [baseOptions, allPermissions, roleId, value]);

  const selectedPermissions = useMemo(() => {
    return value
      .map((id) => {
        // First: check selectedPermissionsData (for pre-loaded edit data)
        const preLoadedPerm = assignedPermsDetails.find((p) => p.id === id);
        if (preLoadedPerm) {
          return preLoadedPerm;
        }

        // Second: check loaded permissions (for newly added permissions)
        const loadedPerm = allPermissions.find((p) => p.id === id);
        if (loadedPerm) {
          return loadedPerm;
        }

        // Third: fallback
        return {
          id: id,
          name: id,
          slug: id,
          description: null,
        };
      })
      .filter((perm): perm is NonNullable<typeof perm> => perm !== undefined);
  }, [value, allPermissions, assignedPermsDetails]);

  const handleRemovePermission = (permissionId: string) => {
    onChange(value.filter((id) => id !== permissionId));
  };

  return (
    <div className="space-y-3">
      <FormCombobox
        optional
        label="Assign permissions"
        description="Select permissions from your workspace."
        options={selectableOptions}
        value=""
        onChange={(e) => setSearchValue(e.currentTarget.value)}
        onSelect={(val) => {
          if (val === "__load_more__") {
            return;
          }
          // Add the selected permission to the array
          if (!value.includes(val)) {
            onChange([...value, val]);
          }
          // Clear search after selection
          setSearchValue("");
        }}
        placeholder={
          <div className="flex w-full text-grayA-8 text-[13px] gap-1.5 items-center py-2">
            Select permissions
          </div>
        }
        searchPlaceholder="Search permissions by name, ID, slug, or description..."
        emptyMessage={
          isSearching ? (
            <div className="px-3 py-3 text-gray-10 text-[13px]">Searching...</div>
          ) : (
            <div className="px-3 py-3 text-gray-10 text-[13px]">No permissions found</div>
          )
        }
        variant="default"
        error={error}
        disabled={disabled}
      />

      {/* Selected Permissions Display */}
      {selectedPermissions.length > 0 && (
        <div className="space-y-2">
          <div className="grid grid-cols-2 gap-2 max-w-[400px]">
            {selectedPermissions.map((permission) => (
              <div
                key={permission.id}
                className="flex items-center gap-2 px-3 py-1.5 bg-white dark:bg-black border border-gray-5 rounded-md text-xs h-12 w-full"
              >
                <div className="border rounded-full flex items-center justify-center border-grayA-6 size-4 flex-shrink-0">
                  <Key2 size="sm-regular" className="text-grayA-11" />
                </div>
                <div className="flex flex-col gap-0.5 min-w-0">
                  <span className="font-medium text-accent-12 truncate text-xs">
                    {permission.name}
                  </span>
                  <span className="text-accent-9 text-[11px] font-mono truncate">
                    {permission.slug}
                  </span>
                </div>
                {!disabled && (
                  <button
                    type="button"
                    onClick={() => handleRemovePermission(permission.id)}
                    className="p-0.5 hover:bg-grayA-4 rounded text-grayA-11 hover:text-accent-12 transition-colors flex-shrink-0 ml-auto"
                    aria-label={`Remove ${permission.name}`}
                  >
                    <XMark size="sm-regular" />
                  </button>
                )}
              </div>
            ))}
          </div>
        </div>
      )}
    </div>
  );
};
