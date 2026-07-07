import { StatusBadge } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/settings/components/status-badge";
import { SelectedItemsList } from "@/components/selected-item-list";
import { FormCombobox } from "@/components/ui/form-combobox";
import type { KeyPermission, KeyRole } from "@/lib/trpc/routers/key/rbac/connected-roles-and-perms";
import { Page2 } from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { useMemo, useState } from "react";
import { useWatch } from "react-hook-form";
import { createPermissionOptions } from "./create-permission-options";
import { useFetchPermissions } from "./hooks/use-fetch-keys-permissions";
import { useSearchPermissions } from "./hooks/use-search-keys-permissions";

type PermissionFieldProps = {
  value: string[];
  onChange: (slugs: string[]) => void;
  error?: string;
  disabled?: boolean;
  assignedPermsDetails: KeyPermission[];
  assignedRoleDetails: KeyRole[];
};

export const PermissionField = ({
  value,
  onChange,
  assignedRoleDetails,
  error,
  disabled = false,
  assignedPermsDetails = [],
}: PermissionFieldProps) => {
  const [searchValue, setSearchValue] = useState("");
  const trimmedSearchVal = searchValue.trim();
  const { permissions, isFetchingNextPage, hasNextPage, loadMore, isLoading } =
    useFetchPermissions();
  const { searchResults, isSearching } = useSearchPermissions(searchValue);

  const selectedRoleNames = useWatch({ name: "roleNames", defaultValue: [] });
  const selectedRoleIds = useMemo(() => {
    return assignedRoleDetails
      .filter((role) => selectedRoleNames.includes(role.name))
      .map((role) => role.id);
  }, [assignedRoleDetails, selectedRoleNames]);

  // Calculate permissions inherited from currently selected roles
  const inheritedPermissions = useMemo(() => {
    const inherited = new Map<string, KeyPermission>();

    assignedPermsDetails.forEach((permission) => {
      if (
        permission.source === "role" &&
        permission.roleId &&
        selectedRoleIds.includes(permission.roleId)
      ) {
        inherited.set(permission.slug, permission);
      }
    });

    return inherited;
  }, [assignedPermsDetails, selectedRoleIds]);

  // All effective permissions (inherited + direct)
  const allEffectivePermissionSlugs = useMemo(() => {
    return new Set([...inheritedPermissions.keys(), ...value]);
  }, [inheritedPermissions, value]);

  // Combine loaded permissions with search results
  const allPermissions = useMemo(() => {
    if (trimmedSearchVal && searchResults.length > 0) {
      return searchResults;
    }
    if (trimmedSearchVal && searchResults.length === 0 && !isSearching) {
      const searchTerm = trimmedSearchVal.toLowerCase();
      return permissions.filter(
        (permission) =>
          permission.id.toLowerCase().includes(searchTerm) ||
          permission.name.toLowerCase().includes(searchTerm) ||
          permission.slug.toLowerCase().includes(searchTerm) ||
          permission.description?.toLowerCase().includes(searchTerm),
      );
    }
    return permissions;
  }, [permissions, searchResults, trimmedSearchVal, isSearching]);

  const showLoadMore = !trimmedSearchVal && hasNextPage;

  const baseOptions = createPermissionOptions({
    permissions: allPermissions,
    hasNextPage: showLoadMore,
    isFetchingNextPage,
    loadMore,
  });

  const selectableOptions = useMemo(() => {
    return baseOptions.filter((option) => {
      if (option.value === "__load_more__") {
        return true;
      }

      // Don't show permissions that are already effective (inherited or direct)
      if (allEffectivePermissionSlugs.has(option.value)) {
        return false;
      }

      return true;
    });
  }, [baseOptions, allEffectivePermissionSlugs]);

  // Combined list for display: inherited permissions + direct permissions
  const displayPermissions = useMemo(() => {
    const permissionsList: Array<KeyPermission & { isInherited: boolean }> = [];

    // Add inherited permissions
    inheritedPermissions.forEach((permission) => {
      permissionsList.push({
        ...permission,
        isInherited: true,
      });
    });

    // Add direct permissions
    value.forEach((permissionSlug) => {
      if (inheritedPermissions.has(permissionSlug)) {
        return;
      }

      const directPermission = assignedPermsDetails.find(
        (p) => p.slug === permissionSlug && p.source === "direct",
      );

      if (directPermission) {
        permissionsList.push({
          ...directPermission,
          isInherited: false,
        });
        return;
      }

      const loadedPerm = allPermissions.find((p) => p.slug === permissionSlug);
      if (loadedPerm) {
        permissionsList.push({
          ...loadedPerm,
          source: "direct" as const,
          isInherited: false,
        });
        return;
      }

      permissionsList.push({
        id: permissionSlug,
        name: permissionSlug,
        slug: permissionSlug,
        description: null,
        source: "direct" as const,
        isInherited: false,
      });
    });

    return permissionsList.sort((a, b) => {
      if (a.isInherited && !b.isInherited) {
        return -1;
      }
      if (!a.isInherited && b.isInherited) {
        return 1;
      }
      return a.name.localeCompare(b.name);
    });
  }, [inheritedPermissions, value, assignedPermsDetails, allPermissions]);

  const handleRemovePermission = (permissionSlug: string) => {
    if (inheritedPermissions.has(permissionSlug)) {
      return;
    }

    onChange(value.filter((slug) => slug !== permissionSlug));
  };

  const handleAddPermission = (permissionSlug: string) => {
    if (allEffectivePermissionSlugs.has(permissionSlug)) {
      return;
    }

    onChange([...value, permissionSlug]);
    setSearchValue("");
  };
  const isComboboxLoading = isLoading || (isSearching && trimmedSearchVal.length > 0);

  return (
    <div className="flex flex-col gap-3">
      <FormCombobox
        label="Assign permissions"
        description="Select permissions from your workspace."
        options={selectableOptions}
        value=""
        onChange={(e) => setSearchValue(e.currentTarget.value)}
        onSelect={handleAddPermission}
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
            <div className="px-3 py-3 text-gray-10 text-[13px]">No permissions found</div>
          )
        }
        variant="default"
        error={error}
        disabled={disabled || isLoading}
        loading={isComboboxLoading}
        title={
          isComboboxLoading
            ? isSearching && trimmedSearchVal
              ? "Searching for permissions..."
              : "Loading available permissions..."
            : undefined
        }
      />

      <SelectedItemsList
        items={displayPermissions.map((permission) => ({ ...permission, id: permission.slug }))}
        disabled={disabled}
        onRemoveItem={handleRemovePermission}
        isItemRemovable={(permission) => !permission.isInherited}
        renderIcon={() => <Page2 iconSize="sm-regular" className="text-grayA-11" />}
        renderPrimaryText={(permission) => permission.name}
        enableTransitions
        renderSecondaryText={(permission) => permission.slug}
        renderBadge={(permission) =>
          permission.isInherited && (
            <InfoTooltip
              className="z-auto"
              variant="primary"
              content={`Inherited from role: ${
                assignedRoleDetails.find((r) => r.id === permission.roleId)?.name
              }`}
            >
              <StatusBadge
                variant="locked"
                text="Inherited via role"
                className="normal-case text-[11px]"
              />
            </InfoTooltip>
          )
        }
      />
    </div>
  );
};
