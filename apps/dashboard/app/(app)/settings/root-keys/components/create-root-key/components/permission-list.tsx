"use client";
import { Collapsible, CollapsibleContent } from "@/components/ui/collapsible";
import type { UnkeyPermission } from "@unkey/rbac";
import { useCallback, useEffect, useMemo, useReducer, useState } from "react";
import { apiPermissions, workspacePermissions } from "../../../[keyId]/permissions/permissions";
import {
  type PermissionCategory,
  type PermissionList,
  computeCheckedStates,
  filterPermissionList,
  getAllPermissionNames,
  permissionReducer,
} from "../utils/permissions";
import { ExpandableCategory } from "./expandable-category";
import { PermissionToggle } from "./permission-toggle";
import { Separator } from "@unkey/ui";

// Utility function to highlight search text
const highlightText = (text: string, searchValue: string | undefined) => {
  if (!searchValue || searchValue.trim() === "") {
    return text;
  }

  const regex = new RegExp(`(${searchValue.replace(/[.*+?^${}()|[\]\\]/g, '\\$&')})`, 'gi');
  const parts = text.split(regex);

  return parts.map((part, index) =>
    regex.test(part) ? (
      <span key={index} className="bg-grayA-4 rounded-[4px] py-0.5">
        {part}
      </span>
    ) : (
      part
    )
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

export const PermissionContentList = ({
  type,
  api,
  onPermissionChange,
  selected,
  searchValue,
}: Props) => {

  const permissionList: PermissionList = useMemo(
    () => (type === "workspace" ? workspacePermissions : api ? apiPermissions(api.id) : {}),
    [type, api?.id],
  );

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

    const { rootChecked, categoryChecked } = computeCheckedStates(initState, filteredPermissionList);

    const selectedPermissions = getAllPermissionNames(filteredPermissionList).filter((permission) =>
      initState.includes(permission),
    );

    return { rootChecked, categoryChecked, selectedPermissions };
  }, [type, selected, filteredPermissionList, api]);

  const [state, dispatch] = useReducer(permissionReducer, {
    selectedPermissions: selected.length > 0 ? initialState.selectedPermissions : [],
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
  }, [filteredPermissionList, state.selectedPermissions]);

  // Notify parent when selectedPermissions changes
  useEffect(() => {
    onPermissionChange(state.selectedPermissions);

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state.selectedPermissions]);

  const handleRootChecked = useCallback(() => {
    dispatch({ type: "TOGGLE_ROOT", permissionList: filteredPermissionList });
  }, [filteredPermissionList]);

  const handleCategoryChecked = useCallback(
    (category: string) => {
      dispatch({ type: "TOGGLE_CATEGORY", category, permissionList: filteredPermissionList });
    },
    [filteredPermissionList],
  );

  const handlePermissionChecked = useCallback(
    (permission: UnkeyPermission) => {

      dispatch({ type: "TOGGLE_PERMISSION", permission, permissionList: filteredPermissionList });
    },
    [filteredPermissionList, searchValue],
  );

  return (
    <div className="flex flex-col max-w-full grow-0">

      <Collapsible
        open={isRootExpanded}
        onOpenChange={setIsRootExpanded}
        className="hover:bg-grayA-3 rounded-lg mb-2">

        <ExpandableCategory
          category={type === "workspace" ? "Workspace" : (api?.name ?? "API")}
          description={
            type === "workspace"
              ? "All workspace permissions"
              : `All permissions for ${api?.name ?? "API"}`
          }
          checked={state.rootChecked}
          setChecked={handleRootChecked}
          count={Object.keys(filteredPermissionList).reduce((acc, category) => acc + Object.keys(filteredPermissionList[category]).length, 0)}
        />

        <CollapsibleContent>

          <div className="flex">
            <div className="flex flex-col pl-5 min-h-full border-r border-grayA-5 mb-2 mr-4">
            </div>
            <div className="flex flex-col h-full w-full">

              {Object.entries(filteredPermissionList).map(([category, allPermissions]) => {
                return (


                  <Collapsible
                    key={`${type === "workspace" ? "workspace" : api?.id}-${category}`}
                    className="w-full rounded-lg justify-center items-center pr-0.5 hover:bg-grayA-3"
                    open={expandedCategories.has(category)}
                    onOpenChange={(open) => {
                      setExpandedCategories(prev => {
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
                      <div className="flex w-full overflow-clip max-w-full ">
                        <div className="flex flex-col pl-5 min-h-full border-r border-grayA-5 mb-2">
                        </div>
                        <div className="flex flex-col h-full max-w-full pl-1 overflow-clip">
                          {Object.entries(allPermissions as PermissionCategory).map(
                            ([action, { description, permission }]) => (
                              <PermissionToggle
                                key={action}
                                category={highlightText(category, searchValue)}
                                label={highlightText(action, searchValue)}
                                description={description}
                                checked={state.selectedPermissions.includes(permission)}
                                setChecked={() => handlePermissionChecked(permission)}
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
