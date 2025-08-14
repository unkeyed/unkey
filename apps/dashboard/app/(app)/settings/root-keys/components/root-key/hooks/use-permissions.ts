import type { UnkeyPermission } from "@unkey/rbac";
import { useCallback, useMemo } from "react";
import { apiPermissions, workspacePermissions } from "../permissions";
import {
  computeCheckedStates,
  filterPermissionList,
  getAllPermissionNames,
} from "../utils/permissions";

type UsePermissionsProps = {
  type: "workspace" | "api";
  api?: { id: string; name: string };
  selected: UnkeyPermission[];
  searchValue?: string;
  onPermissionChange: (permissions: UnkeyPermission[]) => void;
};

export function usePermissions({
  type,
  api,
  selected,
  searchValue,
  onPermissionChange,
}: UsePermissionsProps) {
  // Get permission list based on type
  const permissionList = useMemo(() => {
    if (type === "workspace") {
      return workspacePermissions;
    }
    if (api) {
      return apiPermissions(api.id);
    }
    return {};
  }, [type, api]);

  // Filter permissions based on search
  const filteredPermissionList = useMemo(() => {
    return filterPermissionList(permissionList, searchValue);
  }, [permissionList, searchValue]);

  // Get filtered permissions for this type
  const filteredPermissionsForType = useMemo(() => {
    return getAllPermissionNames(filteredPermissionList);
  }, [filteredPermissionList]);

  // Compute current state based on selected permissions
  const currentSelected = useMemo(() => {
    return getAllPermissionNames(permissionList).filter((permission) =>
      selected.includes(permission),
    );
  }, [permissionList, selected]);

  const { rootChecked, categoryChecked } = useMemo(() => {
    return computeCheckedStates(currentSelected, filteredPermissionList);
  }, [currentSelected, filteredPermissionList]);

  // Event handlers
  const handleRootToggle = useCallback(() => {
    // Get permissions that are NOT in the current filtered view
    const permissionsNotInFilteredView = selected.filter(
      (permission) => !filteredPermissionsForType.includes(permission),
    );

    // Check if all filtered permissions are currently selected
    const allFilteredSelected = filteredPermissionsForType.every((permission) =>
      selected.includes(permission),
    );

    // If all filtered permissions are selected, remove them. Otherwise, add all filtered permissions
    const newFilteredPermissions = allFilteredSelected ? [] : filteredPermissionsForType;

    // Combine non-filtered permissions with the new filtered permissions
    const newSelected = [...permissionsNotInFilteredView, ...newFilteredPermissions];

    onPermissionChange(newSelected);
  }, [filteredPermissionsForType, selected, onPermissionChange]);

  const handleCategoryToggle = useCallback(
    (category: string) => {
      // Get permissions that are NOT in the current filtered view
      const permissionsNotInFilteredView = selected.filter(
        (permission) => !filteredPermissionsForType.includes(permission),
      );

      // Get permissions for this specific category in the filtered view
      const categoryPermissions = Object.values(filteredPermissionList[category] || {}).map(
        ({ permission }) => permission,
      );

      // Check if all category permissions are currently selected
      const allCategorySelected = categoryPermissions.every((p) => selected.includes(p));

      // Get permissions from other categories in the filtered view
      const otherFilteredPermissions = filteredPermissionsForType.filter(
        (permission) => !categoryPermissions.includes(permission) && selected.includes(permission),
      );

      // If all category permissions are selected, remove them. Otherwise, add all category permissions
      const newCategoryPermissions = allCategorySelected ? [] : categoryPermissions;

      // Combine all permissions
      const newSelected = [
        ...permissionsNotInFilteredView,
        ...otherFilteredPermissions,
        ...newCategoryPermissions,
      ];

      onPermissionChange(newSelected);
    },
    [filteredPermissionList, filteredPermissionsForType, selected, onPermissionChange],
  );

  const handlePermissionToggle = useCallback(
    (permission: UnkeyPermission) => {
      const isSelected = currentSelected.includes(permission);
      const newSelected = isSelected
        ? currentSelected.filter((p) => p !== permission)
        : [...currentSelected, permission];
      onPermissionChange(newSelected);
    },
    [currentSelected, onPermissionChange],
  );

  return {
    state: {
      selectedPermissions: currentSelected,
      rootChecked,
      categoryChecked,
    },
    filteredPermissionList,
    handleRootToggle,
    handleCategoryToggle,
    handlePermissionToggle,
  };
}
