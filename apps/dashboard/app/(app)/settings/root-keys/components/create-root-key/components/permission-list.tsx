"use client";
import { Collapsible, CollapsibleContent } from "@/components/ui/collapsible";
import type { CheckedState } from "@radix-ui/react-checkbox";
import type { UnkeyPermission } from "@unkey/rbac";
import { useEffect, useReducer } from "react";
import { apiPermissions, workspacePermissions } from "../../../[keyId]/permissions/permissions";
import { ExpandableCategory } from "./expandable-category";
import { PermissionToggle } from "./permission-toggle";

type Props = {
  type: "workspace" | "api";
  api?:
    | {
        id: string;
        name: string;
      }
    | undefined;
  onPermissionChange: (permissions: UnkeyPermission[]) => void;
};

// Helper type for permission list structure
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
  | { type: "TOGGLE_ROOT"; permissionList: PermissionList };

function getAllPermissionNames(permissionList: PermissionList) {
  return Object.values(permissionList).flatMap((category) =>
    Object.values(category).map(({ permission }) => permission),
  );
}

function getCategoryPermissionNames(permissionList: PermissionList, category: string) {
  return Object.values(permissionList[category] || {}).map(({ permission }) => permission);
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
    default:
      return state;
  }
}

export const PermissionContentList = ({ type, api, onPermissionChange }: Props) => {
  const permissionList: PermissionList =
    type === "workspace" ? workspacePermissions : api ? apiPermissions(api.id) : {};

  const [state, dispatch] = useReducer(permissionReducer, {
    selectedPermissions: [],
    categoryChecked: {},
    rootChecked: false,
  });

  // Keep checked states in sync with permissionList changes (e.g. api switch)
  useEffect(() => {
    const { rootChecked, categoryChecked } = computeCheckedStates(
      state.selectedPermissions,
      permissionList,
    );
    if (
      rootChecked !== state.rootChecked ||
      JSON.stringify(categoryChecked) !== JSON.stringify(state.categoryChecked)
    ) {
      // Only update if out of sync
      dispatch({ type: "TOGGLE_ROOT", permissionList });
    }
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [permissionList]);

  // Notify parent when selectedPermissions changes
  useEffect(() => {
    onPermissionChange(state.selectedPermissions);
    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state.selectedPermissions]);

  const handleRootChecked = () => {
    dispatch({ type: "TOGGLE_ROOT", permissionList });
  };

  const handleCategoryChecked = (category: string) => {
    dispatch({ type: "TOGGLE_CATEGORY", category, permissionList });
  };

  const handlePermissionChecked = (permission: UnkeyPermission) => {
    dispatch({ type: "TOGGLE_PERMISSION", permission, permissionList });
  };

  return (
    <div className="flex flex-col gap-2">
      <Collapsible>
        <ExpandableCategory
          category={type === "workspace" ? "Workspace" : (api?.name ?? "API")}
          description={
            type === "workspace"
              ? "All workspace permissions"
              : `All permissions for ${api?.name ?? "API"}`
          }
          checked={state.rootChecked}
          setChecked={handleRootChecked}
        />
        <CollapsibleContent>
          <div className="flex flex-col">
            {Object.entries(permissionList).map(([category, allPermissions]) => {
              return (
                <>
                  <div
                    key={`${type === "workspace" ? "workspace" : api?.id}-${category}`}
                    className="flex flex-col gap-2 my-0 py-0 border-l ml-[31px]"
                  >
                    <div className="flex flex-col my-0 py-0">
                      <Collapsible>
                        <ExpandableCategory
                          category={category}
                          checked={state.categoryChecked[category]}
                          description={""}
                          setChecked={() => handleCategoryChecked(category)}
                        />
                        <CollapsibleContent>
                          <div className="flex flex-col gap-2 my-0 py-0 border-l ml-[31px]">
                            {Object.entries(allPermissions as PermissionCategory).map(
                              ([action, { description, permission }]) => (
                                <PermissionToggle
                                  key={action}
                                  category={category}
                                  label={action}
                                  description={description}
                                  checked={state.selectedPermissions.includes(permission)}
                                  setChecked={() => handlePermissionChecked(permission)}
                                />
                              ),
                            )}
                          </div>
                        </CollapsibleContent>
                      </Collapsible>
                    </div>
                  </div>
                </>
              );
            })}
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
};
