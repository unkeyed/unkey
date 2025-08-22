import type { CheckedState } from "@radix-ui/react-checkbox";
import type { UnkeyPermission } from "@unkey/rbac";

// Type definitions
export type PermissionItem = {
  description: string;
  permission: UnkeyPermission;
};

export type PermissionCategory = Record<string, PermissionItem>;
export type PermissionList = Record<string, PermissionCategory>;

// State management
export interface PermissionState {
  selectedPermissions: UnkeyPermission[];
  categoryChecked: Record<string, CheckedState>;
  rootChecked: CheckedState;
}

export type PermissionAction =
  | {
      type: "TOGGLE_PERMISSION";
      permission: UnkeyPermission;
      permissionList: PermissionList;
    }
  | {
      type: "TOGGLE_CATEGORY";
      category: string;
      permissionList: PermissionList;
    }
  | { type: "TOGGLE_ROOT"; permissionList: PermissionList }
  | {
      type: "UPDATE_CHECKED_STATES";
      rootChecked: CheckedState;
      categoryChecked: Record<string, CheckedState>;
    }
  | {
      type: "UPDATE_SELECTED_PERMISSIONS";
      selectedPermissions: UnkeyPermission[];
    };

// Utility functions
export function getAllPermissionNames(permissionList: PermissionList): UnkeyPermission[] {
  return Object.values(permissionList).flatMap((category) =>
    Object.values(category).map(({ permission }) => permission),
  );
}

export function getCategoryPermissionNames(
  permissionList: PermissionList,
  category: string,
): UnkeyPermission[] {
  return Object.values(permissionList[category] || {}).map(({ permission }) => permission);
}

export function filterPermissionList(
  permissionList: PermissionList,
  searchValue: string | undefined,
): PermissionList {
  if (!searchValue || searchValue.trim() === "") {
    return permissionList;
  }

  const searchLower = searchValue.toLowerCase();

  return Object.entries(permissionList).reduce<PermissionList>((acc, [category, permissions]) => {
    const filteredPermissions = Object.entries(permissions).reduce<Record<string, PermissionItem>>(
      (acc, [permissionName, { description, permission }]) => {
        const permissionNameLower = permissionName.toLowerCase();
        const wordMatched = permissionNameLower.includes(searchLower);

        if (wordMatched) {
          acc[permissionName] = { description, permission };
        }
        return acc;
      },
      {},
    );

    if (Object.keys(filteredPermissions).length > 0) {
      acc[category] = filteredPermissions;
    }
    return acc;
  }, {});
}

export function computeCheckedStates(
  selectedPermissions: UnkeyPermission[],
  permissionList: PermissionList,
): { rootChecked: CheckedState; categoryChecked: Record<string, CheckedState> } {
  const allPermissionNames = getAllPermissionNames(permissionList);

  // Compute root checked state
  let rootChecked: CheckedState = false;
  if (selectedPermissions.length === 0) {
    rootChecked = false;
  } else if (selectedPermissions.length === allPermissionNames.length) {
    rootChecked = true;
  } else {
    rootChecked = "indeterminate";
  }

  // Compute category checked states
  const categoryChecked: Record<string, CheckedState> = {};
  Object.entries(permissionList).forEach(([category, allPermissions]) => {
    const categoryPermissions = Object.values(allPermissions).map(({ permission }) => permission);
    const selectedInCategory = categoryPermissions.filter((p) => selectedPermissions.includes(p));

    if (selectedInCategory.length === 0) {
      categoryChecked[category] = false;
    } else if (selectedInCategory.length === categoryPermissions.length) {
      categoryChecked[category] = true;
    } else {
      categoryChecked[category] = "indeterminate";
    }
  });

  return { rootChecked, categoryChecked };
}

export function permissionReducer(
  state: PermissionState,
  action: PermissionAction,
): PermissionState {
  switch (action.type) {
    case "TOGGLE_PERMISSION": {
      const { permission, permissionList } = action;
      const isSelected = state.selectedPermissions.includes(permission);
      const selectedPermissions = isSelected
        ? state.selectedPermissions.filter((p) => p !== permission)
        : [...state.selectedPermissions, permission];

      const { rootChecked, categoryChecked } = computeCheckedStates(
        selectedPermissions,
        permissionList,
      );
      return { selectedPermissions, rootChecked, categoryChecked };
    }

    case "TOGGLE_CATEGORY": {
      const { category, permissionList } = action;
      const categoryPermissions = getCategoryPermissionNames(permissionList, category);
      const allSelected = categoryPermissions.every((p) => state.selectedPermissions.includes(p));

      const selectedPermissions = allSelected
        ? state.selectedPermissions.filter((p) => !categoryPermissions.includes(p))
        : Array.from(new Set([...state.selectedPermissions, ...categoryPermissions]));

      const { rootChecked, categoryChecked } = computeCheckedStates(
        selectedPermissions,
        permissionList,
      );
      return { selectedPermissions, rootChecked, categoryChecked };
    }

    case "TOGGLE_ROOT": {
      const { permissionList } = action;
      const allPermissionNames = getAllPermissionNames(permissionList);
      const allSelected = state.selectedPermissions.length === allPermissionNames.length;

      const selectedPermissions = allSelected ? [] : allPermissionNames;
      const { rootChecked, categoryChecked } = computeCheckedStates(
        selectedPermissions,
        permissionList,
      );
      return { selectedPermissions, rootChecked, categoryChecked };
    }

    case "UPDATE_CHECKED_STATES": {
      const { rootChecked, categoryChecked } = action;
      return { ...state, rootChecked, categoryChecked };
    }

    case "UPDATE_SELECTED_PERMISSIONS": {
      const { selectedPermissions } = action;
      return { ...state, selectedPermissions };
    }

    default:
      return state;
  }
}

export function hasPermissionResults(
  permissionList: PermissionList,
  searchValue?: string,
): boolean {
  if (!searchValue || searchValue.trim() === "") {
    return Object.keys(permissionList).some(
      (category) => Object.keys(permissionList[category]).length > 0,
    );
  }

  const searchLower = searchValue.toLowerCase();
  return Object.values(permissionList).some((category: PermissionCategory) =>
    Object.keys(category).some((permissionName) =>
      permissionName.toLowerCase().includes(searchLower),
    ),
  );
}
