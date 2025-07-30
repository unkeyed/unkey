"use client";
import { Collapsible, CollapsibleContent } from "@/components/ui/collapsible";
// Local utility functions
import type { CheckedState } from "@radix-ui/react-checkbox";
import type { UnkeyPermission } from "@unkey/rbac";
import { useCallback, useEffect, useMemo, useReducer, useRef, useState } from "react";
import { apiPermissions, workspacePermissions } from "../../../[keyId]/permissions/permissions";
import { ExpandableCategory } from "./expandable-category";
import { PermissionToggle } from "./permission-toggle";

// Type definitions for permission structure (same as permission-sheet)
type PermissionItem = {
  description: string;
  permission: string; // Using string instead of UnkeyPermission to handle literal types
};
type PermissionCategory = Record<string, PermissionItem>;
type PermissionList = Record<string, PermissionCategory>;

// Utility function to highlight search text
const highlightText = (text: string, searchValue: string | undefined) => {
  if (!searchValue || searchValue.trim() === "") {
    return text;
  }

  const regex = new RegExp(`(${searchValue.replace(/[.*+?^${}()|[\]\\]/g, "\\$&")})`, "gi");
  const parts = text.split(regex);

  return parts.map((part, index) =>
    regex.test(part) ? (
      <span key={index + part} className="bg-grayA-4 rounded-[4px] py-0.5">
        {part}
      </span>
    ) : (
      part
    ),
  );
};

type Props = {
  type: "workspace" | "api";
  api?:
    | {
        id: string;
        name: string;
      }
    | undefined;
  onPermissionChange: (permissions: UnkeyPermission[]) => void;
  selected: UnkeyPermission[];
  searchValue: string | undefined;
};

// Local utility functions
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
      Record<string, { description: string; permission: string }>
    >(
      (acc, [permissionName, { description, permission }]) => {
        if (searchValue && searchValue.length > 0) {
          const permissionNameLower = permissionName.toLowerCase();
          const wordMatched = permissionNameLower.includes(searchValue.toLowerCase());
          if (!wordMatched) {
            return acc;
          }
        }
        acc[permissionName] = { description, permission };
        return acc;
      },
      {} as Record<string, { description: string; permission: string }>,
    );
    return acc as PermissionList;
  }, {} as PermissionList);
}

function computeCheckedStates(
  selectedPermissions: UnkeyPermission[],
  permissionList: PermissionList,
) {
  const allPermissionNames = getAllPermissionNames(permissionList);
  let rootChecked: CheckedState = false;

  if (selectedPermissions.length === 0) {
    rootChecked = false;
  } else if (selectedPermissions.length === allPermissionNames.length) {
    rootChecked = true;
  } else {
    rootChecked = "indeterminate";
  }

  const categoryChecked: Record<string, CheckedState> = {};

  Object.entries(permissionList).forEach(([category, allPermissions]) => {
    const allPermissionNames = Object.values(allPermissions).map(({ permission }) => permission);
    if (allPermissionNames.every((p) => selectedPermissions.includes(p as UnkeyPermission))) {
      categoryChecked[category] = true;
    } else if (allPermissionNames.some((p) => selectedPermissions.includes(p as UnkeyPermission))) {
      categoryChecked[category] = "indeterminate";
    } else {
      categoryChecked[category] = false;
    }
  });

  return { rootChecked, categoryChecked };
}

// State and actions for reducer
interface PermissionState {
  selectedPermissions: UnkeyPermission[];
  categoryChecked: Record<string, CheckedState>;
  rootChecked: CheckedState;
}

type PermissionAction =
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
    };

function permissionReducer(state: PermissionState, action: PermissionAction): PermissionState {
  switch (action.type) {
    case "TOGGLE_PERMISSION": {
      const { permission, permissionList } = action;
      let selectedPermissions: UnkeyPermission[];
      if (state.selectedPermissions.includes(permission as UnkeyPermission)) {
        selectedPermissions = state.selectedPermissions.filter(
          (p) => p !== (permission as UnkeyPermission),
        );
      } else {
        selectedPermissions = [...state.selectedPermissions, permission as UnkeyPermission];
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
      const allSelected = categoryPermissions.every((p) =>
        state.selectedPermissions.includes(p as UnkeyPermission),
      );
      let selectedPermissions: UnkeyPermission[];
      if (allSelected) {
        selectedPermissions = state.selectedPermissions.filter(
          (p) => !categoryPermissions.includes(p as string),
        );
      } else {
        selectedPermissions = Array.from(
          new Set([...state.selectedPermissions, ...(categoryPermissions as UnkeyPermission[])]),
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
        selectedPermissions = allPermissionNames as UnkeyPermission[];
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

// Helper function to convert workspace permissions to PermissionList type
const getPermissionList = (
  type: "workspace" | "api",
  api?: { id: string; name: string },
): PermissionList => {
  if (type === "workspace") {
    return workspacePermissions;
  }
  if (api) {
    return apiPermissions(api.id);
  }
  return {};
};

export const PermissionContentList = ({
  type,
  api,
  onPermissionChange,
  selected,
  searchValue,
}: Props) => {
  // Use ref to store the latest callback to prevent infinite loops
  const onPermissionChangeRef = useRef(onPermissionChange);

  // Update ref when callback changes
  useEffect(() => {
    onPermissionChangeRef.current = onPermissionChange;
  }, [onPermissionChange]);
  const permissionList = useMemo(() => getPermissionList(type, api), [type, api]);

  const filteredPermissionList = useMemo(() => {
    if (searchValue !== undefined && searchValue !== "") {
      return filterPermissionList(permissionList, searchValue);
    }
    return permissionList;
  }, [permissionList, searchValue]);

  // State to track expanded categories and root
  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(new Set());
  const [isRootExpanded, setIsRootExpanded] = useState(false);

  // Auto-expand when search is active
  useEffect(() => {
    if (searchValue && searchValue.trim() !== "") {
      setIsRootExpanded(true);
      setExpandedCategories(new Set(Object.keys(filteredPermissionList)));
    } else {
      setIsRootExpanded(false);
      setExpandedCategories(new Set());
    }
  }, [searchValue, filteredPermissionList]);

  const initialState = useMemo(() => {
    const initState =
      type === "workspace"
        ? Object.values(workspacePermissions)
            .flatMap((category) =>
              Object.values(category).map(({ permission }) =>
                selected.includes(permission) ? permission : null,
              ),
            )
            .filter((permission) => permission !== null)
        : api
          ? Object.values(apiPermissions(api.id))
              .flatMap((category) =>
                Object.values(category).map(({ permission }) =>
                  selected.includes(permission) ? permission : null,
                ),
              )
              .filter((permission) => permission !== null)
          : [];

    const { rootChecked, categoryChecked } = computeCheckedStates(initState, permissionList);

    const selectedPermissions = getAllPermissionNames(permissionList).filter((permission) =>
      initState.includes(permission),
    );

    return { rootChecked, categoryChecked, selectedPermissions };
  }, [type, selected, permissionList, api]);

  const [state, dispatch] = useReducer(permissionReducer, {
    selectedPermissions:
      selected.length > 0 ? (initialState.selectedPermissions as UnkeyPermission[]) : [],
    categoryChecked: selected.length > 0 ? initialState.categoryChecked : {},
    rootChecked: selected.length > 0 ? initialState.rootChecked : false,
  });

  // Recalculate checked states when filteredPermissionList changes (due to search)
  useEffect(() => {
    const { rootChecked, categoryChecked } = computeCheckedStates(
      state.selectedPermissions,
      filteredPermissionList,
    );

    // Only update if the checked states have actually changed
    if (
      rootChecked !== state.rootChecked ||
      JSON.stringify(categoryChecked) !== JSON.stringify(state.categoryChecked)
    ) {
      dispatch({
        type: "UPDATE_CHECKED_STATES",
        rootChecked,
        categoryChecked,
      });
    }
  }, [filteredPermissionList, state.selectedPermissions, state.rootChecked, state.categoryChecked]);

  // Notify parent when selectedPermissions changes
  useEffect(() => {
    onPermissionChangeRef.current(state.selectedPermissions);
  }, [state.selectedPermissions]);

  const handleRootChecked = useCallback(() => {
    dispatch({ type: "TOGGLE_ROOT", permissionList: filteredPermissionList });
  }, [filteredPermissionList]);

  const handleCategoryChecked = useCallback(
    (category: string) => {
      dispatch({
        type: "TOGGLE_CATEGORY",
        category,
        permissionList: filteredPermissionList,
      });
    },
    [filteredPermissionList],
  );

  const handlePermissionChecked = useCallback(
    (permission: UnkeyPermission) => {
      dispatch({
        type: "TOGGLE_PERMISSION",
        permission,
        permissionList: filteredPermissionList,
      });
    },
    [filteredPermissionList],
  );

  return (
    <div className="flex flex-col max-w-full grow-0">
      <Collapsible
        open={isRootExpanded}
        onOpenChange={setIsRootExpanded}
        className="hover:bg-grayA-3 rounded-lg mb-2"
      >
        <ExpandableCategory
          category={type === "workspace" ? "Workspace" : (api?.name ?? "API")}
          description={
            type === "workspace"
              ? "All workspace permissions"
              : `All permissions for ${api?.name ?? "API"}`
          }
          checked={state.rootChecked}
          setChecked={handleRootChecked}
          count={Object.keys(filteredPermissionList).reduce(
            (acc, category) => acc + Object.keys(filteredPermissionList[category]).length,
            0,
          )}
        />

        <CollapsibleContent>
          <div className="flex">
            <div className="flex flex-col pl-5 min-h-full border-r border-grayA-5 mb-2 mr-4" />
            <div className="flex flex-col h-full w-full">
              {Object.entries(filteredPermissionList).map(([category, allPermissions]) => {
                return (
                  <Collapsible
                    key={`${type === "workspace" ? "workspace" : api?.id}-${category}`}
                    className="w-full rounded-lg justify-center items-center pr-0.5 hover:bg-grayA-3"
                    open={expandedCategories.has(category)}
                    onOpenChange={(open) => {
                      setExpandedCategories((prev) => {
                        const newSet = new Set(prev);
                        if (open) {
                          newSet.add(category);
                        } else {
                          newSet.delete(category);
                        }
                        return newSet;
                      });
                    }}
                  >
                    <ExpandableCategory
                      category={category}
                      checked={state.categoryChecked[category]}
                      description={""}
                      setChecked={() => handleCategoryChecked(category)}
                      count={Object.keys(allPermissions).length}
                    />
                    <CollapsibleContent>
                      <div className="flex w-full justify-start items-start overflow-clip">
                        <div className="flex flex-col pl-5 min-h-full border-r border-grayA-5 mb-2" />
                        <div className="flex flex-col w-full justify-start items-start">
                          {Object.entries(allPermissions as PermissionCategory).map(
                            ([action, { description, permission }]) => (
                              <PermissionToggle
                                key={action}
                                category={highlightText(category, searchValue)}
                                label={highlightText(action, searchValue)}
                                description={description}
                                checked={state.selectedPermissions.includes(
                                  permission as UnkeyPermission,
                                )}
                                setChecked={() =>
                                  handlePermissionChecked(permission as UnkeyPermission)
                                }
                              />
                            ),
                          )}
                        </div>
                      </div>
                    </CollapsibleContent>
                  </Collapsible>
                );
              })}
            </div>
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
};
