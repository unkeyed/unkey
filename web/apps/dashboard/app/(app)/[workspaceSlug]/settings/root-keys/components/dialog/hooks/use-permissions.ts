import type { UnkeyPermission } from "@unkey/rbac";
import { useCallback, useMemo } from "react";
import { type PermissionScope, getScopedPermissions } from "../permissions";
import {
  computeCheckedStates,
  filterPermissionList,
  getAllPermissionNames,
} from "../utils/permissions";

type UsePermissionsProps = {
  scope: PermissionScope;
  selected: UnkeyPermission[];
  searchValue?: string;
  onPermissionChange: (permissions: UnkeyPermission[]) => void;
};

export function usePermissions({
  scope,
  selected,
  searchValue,
  onPermissionChange,
}: UsePermissionsProps) {
  const permissionList = useMemo(() => getScopedPermissions(scope), [scope]);

  const filteredPermissionList = useMemo(() => {
    return filterPermissionList(permissionList, searchValue);
  }, [permissionList, searchValue]);

  const filteredPermissionsForType = useMemo(() => {
    return getAllPermissionNames(filteredPermissionList);
  }, [filteredPermissionList]);

  const currentSelected = useMemo(() => {
    return getAllPermissionNames(permissionList).filter((permission) =>
      selected.includes(permission),
    );
  }, [permissionList, selected]);

  const { rootChecked, categoryChecked } = useMemo(() => {
    return computeCheckedStates(currentSelected, filteredPermissionList);
  }, [currentSelected, filteredPermissionList]);

  const handleRootToggle = useCallback(() => {
    const permissionsNotInFilteredView = selected.filter(
      (permission) => !filteredPermissionsForType.includes(permission),
    );

    const allFilteredSelected = filteredPermissionsForType.every((permission) =>
      selected.includes(permission),
    );

    const newFilteredPermissions = allFilteredSelected ? [] : filteredPermissionsForType;
    const newSelected = [...permissionsNotInFilteredView, ...newFilteredPermissions];

    onPermissionChange(newSelected);
  }, [filteredPermissionsForType, selected, onPermissionChange]);

  const handleCategoryToggle = useCallback(
    (category: string) => {
      const permissionsNotInFilteredView = selected.filter(
        (permission) => !filteredPermissionsForType.includes(permission),
      );

      const categoryPermissions = Object.values(filteredPermissionList[category] || {}).map(
        ({ permission }) => permission,
      );

      const allCategorySelected = categoryPermissions.every((p) => selected.includes(p));

      const otherFilteredPermissions = filteredPermissionsForType.filter(
        (permission) => !categoryPermissions.includes(permission) && selected.includes(permission),
      );

      const newCategoryPermissions = allCategorySelected ? [] : categoryPermissions;

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
      const isSelected = selected.includes(permission);
      const newSelected = isSelected
        ? selected.filter((p) => p !== permission)
        : [...selected, permission];
      onPermissionChange(newSelected);
    },
    [selected, onPermissionChange],
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
