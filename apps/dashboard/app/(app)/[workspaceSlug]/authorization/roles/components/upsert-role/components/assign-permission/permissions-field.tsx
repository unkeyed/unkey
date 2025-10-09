import { SelectedItemsList } from "@/components/selected-item-list";
import { FormCombobox } from "@/components/ui/form-combobox";
import type { RolePermission } from "@/lib/trpc/routers/authorization/roles/connected-keys-and-perms";
import { Page2 } from "@unkey/icons";
import { useMemo, useState } from "react";
import { useRoleLimits } from "../../../table/hooks/use-role-limits";
import { RoleWarningCallout } from "../warning-callout";
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
  assignedPermsDetails,
}: PermissionFieldProps) => {
  const [searchValue, setSearchValue] = useState("");

  const { calculateLimits } = useRoleLimits(roleId);
  const { hasPermWarning, totalPerms } = calculateLimits(value);

  const { permissions, isFetchingNextPage, hasNextPage, loadMore, isLoading } =
    useFetchPermissions();
  const { searchResults, isSearching } = useSearchPermissions(searchValue);

  // Combine loaded permissions with search results, prioritizing search when available
  const allPermissions = useMemo(() => {
    if (searchValue.trim() && searchResults.length > 0) {
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
          permission.description?.toLowerCase().includes(searchTerm)
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
      if (option.value === "__load_more__") {
        return true;
      }
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
        return !permission.roles?.some((role) => role.id === roleId);
      }
      return true;
    });
  }, [baseOptions, allPermissions, roleId, value]);

  const selectedPermissions = useMemo(() => {
    return value
      .map((permId) => {
        // First: check selectedPermissionsData (for pre-loaded edit data)
        const preLoadedPerm = assignedPermsDetails.find((p) => p.id === permId);
        if (preLoadedPerm) {
          return {
            id: preLoadedPerm.id,
            name: preLoadedPerm.name,
            slug: preLoadedPerm.slug,
          };
        }

        // Second: check loaded permissions (for newly added permissions)
        const loadedPerm = allPermissions.find((p) => p.id === permId);
        if (loadedPerm) {
          return loadedPerm;
        }

        // Third: fallback
        return {
          id: permId,
          name: null,
          slug: null,
        };
      })
      .filter((perm): perm is NonNullable<typeof perm> => perm !== undefined);
  }, [value, allPermissions, assignedPermsDetails]);

  const handleRemovePermission = (permissionId: string) => {
    onChange(value.filter((id) => id !== permissionId));
  };

  const handleAddPermission = (permissionId: string) => {
    if (!value.includes(permissionId)) {
      onChange([...value, permissionId]);
    }
    setSearchValue("");
  };

  const isComboboxLoading =
    isLoading || (isSearching && searchValue.trim().length > 0);

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
          handleAddPermission(val);
        }}
        placeholder={
          <div className="flex w-full text-grayA-8 text-[13px] gap-1.5 items-center py-2">
            Select permissions
          </div>
        }
        searchPlaceholder="Search permissions by name, ID, slug, or description..."
        emptyMessage={
          isComboboxLoading ? (
            <div className="px-3 py-3 text-gray-10 text-[13px] flex items-center gap-2">
              <div className="animate-spin h-3 w-3 border border-gray-6 border-t-gray-11 rounded-full" />
              {isSearching ? "Searching..." : "Loading permissions..."}
            </div>
          ) : (
            <div className="px-3 py-3 text-gray-10 text-[13px]">
              No permissions found
            </div>
          )
        }
        variant="default"
        error={error}
        disabled={disabled || isLoading || hasPermWarning}
        loading={isComboboxLoading}
        title={
          isComboboxLoading
            ? isSearching && searchValue.trim()
              ? "Searching for permissions..."
              : "Loading available permissions..."
            : undefined
        }
      />
      {hasPermWarning ? (
        <RoleWarningCallout count={totalPerms} type="permissions" />
      ) : (
        selectedPermissions.length > 0 && (
          <SelectedItemsList
            items={selectedPermissions.map((k) => ({
              ...k,
              name: k.name ?? "Unnamed Permission",
            }))}
            disabled={disabled}
            onRemoveItem={handleRemovePermission}
            renderIcon={() => (
              <Page2 iconSize="sm-regular" className="text-grayA-11" />
            )}
            renderPrimaryText={(permission) => permission.name}
            enableTransitions
            // This can't cannot happen but we need it to make TS happy
            renderSecondaryText={(permission) =>
              permission.slug ?? "Unnamed Slug"
            }
          />
        )
      )}
    </div>
  );
};
