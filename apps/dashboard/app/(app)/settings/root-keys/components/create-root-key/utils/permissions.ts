import type { CheckedState } from "@radix-ui/react-checkbox";
import type { UnkeyPermission } from "@unkey/rbac";

type PermissionCategory = Record<string, { description: string; permission: UnkeyPermission }>;
type PermissionList = Record<string, PermissionCategory>;

// State and actions for reducer
interface PermissionState {
    selectedPermissions: UnkeyPermission[];
    categoryChecked: Record<string, CheckedState>;
    rootChecked: CheckedState;
}

type PermissionAction =
    | { type: "TOGGLE_PERMISSION"; permission: UnkeyPermission; permissionList: PermissionList }
    | { type: "TOGGLE_CATEGORY"; category: string; permissionList: PermissionList }
    | { type: "TOGGLE_ROOT"; permissionList: PermissionList }
    | { type: "UPDATE_CHECKED_STATES"; rootChecked: CheckedState; categoryChecked: Record<string, CheckedState> };

function getAllPermissionNames(permissionList: PermissionList) {
    return Object.values(permissionList).flatMap((category) =>
        Object.values(category).map(({ permission }) => permission),
    );
}

function getCategoryPermissionNames(permissionList: PermissionList, category: string) {
    return Object.values(permissionList[category] || {}).map(({ permission }) => permission);
}


function filterPermissionList(permissionList: PermissionList, searchValue: string | undefined) {
    return Object.entries(permissionList).reduce<PermissionList>((acc, [category, permissions]) => {
        acc[category] = Object.entries(permissions).reduce<
            Record<string, { description: string; permission: UnkeyPermission }>
        >(
            (acc, [permissionName, { description, permission }]) => {
                
                if (searchValue && searchValue.length > 0) {
                    const permissionNameLower = permissionName.toLowerCase();
                    // Check if each word in parsedSearchValue is included in the description
                    const wordMatched = permissionNameLower.includes(searchValue.toLowerCase());
                    if (!wordMatched) {
                        return acc;
                    }
                }
                acc[permissionName] = { description, permission };
                return acc;
            },
            {} as Record<string, { description: string; permission: UnkeyPermission }>,
        );
        return acc as PermissionList;
    }, {} as PermissionList);
}

function computeCheckedStates(
    selectedPermissions: UnkeyPermission[],
    permissionList: PermissionList,
) {
    // Compute rootChecked
    const allPermissionNames = getAllPermissionNames(permissionList);
    let rootChecked: CheckedState = false;

    if (selectedPermissions.length === 0) {
        rootChecked = false;
    } else if (selectedPermissions.length === allPermissionNames.length) {
        rootChecked = true;
    } else {
        rootChecked = "indeterminate";
    }
    // Compute categoryChecked
    const categoryChecked: Record<string, CheckedState> = {};

    Object.entries(permissionList).forEach(([category, allPermissions]) => {
        const allPermissionNames = Object.values(allPermissions).map(({ permission }) => permission);
        if (allPermissionNames.every((p) => selectedPermissions.includes(p))) {
            categoryChecked[category] = true;
        } else if (allPermissionNames.some((p) => selectedPermissions.includes(p))) {
            categoryChecked[category] = "indeterminate";
        } else {
            categoryChecked[category] = false;
        }
    });

    return { rootChecked, categoryChecked };
}

function permissionReducer(state: PermissionState, action: PermissionAction): PermissionState {
    switch (action.type) {
        case "TOGGLE_PERMISSION": {
            const { permission, permissionList } = action;
            let selectedPermissions: UnkeyPermission[];
            if (state.selectedPermissions.includes(permission)) {
                selectedPermissions = state.selectedPermissions.filter((p) => p !== permission);
            } else {
                selectedPermissions = [...state.selectedPermissions, permission];
            }
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
            let selectedPermissions: UnkeyPermission[];
            if (allSelected) {
                // Remove all permissions in this category
                selectedPermissions = state.selectedPermissions.filter(
                    (p) => !categoryPermissions.includes(p),
                );
            } else {
                // Add all permissions in this category
                selectedPermissions = Array.from(
                    new Set([...state.selectedPermissions, ...categoryPermissions]),
                );
            }
            const { rootChecked, categoryChecked } = computeCheckedStates(
                selectedPermissions,
                permissionList,
            );
            return { selectedPermissions, rootChecked, categoryChecked };
        }

        case "TOGGLE_ROOT": {
            const { permissionList } = action;
            const allPermissionNames = getAllPermissionNames(permissionList);
            let selectedPermissions: UnkeyPermission[];
            if (state.selectedPermissions.length === allPermissionNames.length) {
                selectedPermissions = [];
            } else {
                selectedPermissions = allPermissionNames;
            }
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

        default:
            return state;
    }
}

export {
    type PermissionState,
    type PermissionAction,
    type PermissionList,
    type PermissionCategory,
    getAllPermissionNames,
    getCategoryPermissionNames,
    computeCheckedStates,
    permissionReducer,
    filterPermissionList,
};
