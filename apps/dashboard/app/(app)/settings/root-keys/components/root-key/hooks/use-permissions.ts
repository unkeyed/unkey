import type { UnkeyPermission } from "@unkey/rbac";
import { useCallback, useMemo } from "react";
import { apiPermissions, workspacePermissions } from "../../../[keyId]/permissions/permissions";
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
    const allPermissionNames = getAllPermissionNames(filteredPermissionList);
    const allSelected = currentSelected.length === allPermissionNames.length;
    const newSelected = allSelected ? [] : allPermissionNames;
    onPermissionChange(newSelected);
  }, [filteredPermissionList, currentSelected, onPermissionChange]);

  const handleCategoryToggle = useCallback(
    (category: string) => {
      const categoryPermissions = Object.values(filteredPermissionList[category] || {}).map(
        ({ permission }) => permission,
      );
      const allSelected = categoryPermissions.every((p) => currentSelected.includes(p));

      const newSelected = allSelected
        ? currentSelected.filter((p) => !categoryPermissions.includes(p))
        : Array.from(new Set([...currentSelected, ...categoryPermissions]));

      onPermissionChange(newSelected);
    },
    [filteredPermissionList, currentSelected, onPermissionChange],
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
