"use client";
import { Collapsible, CollapsibleContent } from "@/components/ui/collapsible";
import type { UnkeyPermission } from "@unkey/rbac";
import { useCallback, useEffect, useMemo, useReducer } from "react";
import { apiPermissions, workspacePermissions } from "../../../[keyId]/permissions/permissions";
import {
  type PermissionCategory,
  type PermissionList,
  computeCheckedStates,
  filterCategory,
  filterPermissionList,
  getAllPermissionNames,
  permissionReducer,
} from "../utils/permissions";
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
  selected: UnkeyPermission[];
  searchValue: string | undefined;
  setEmpty: (val: boolean) => void;
};

export const PermissionContentList = ({
  type,
  api,
  onPermissionChange,
  selected,
  searchValue,
  setEmpty,
}: Props) => {
  const permissionList: PermissionList = useMemo(
    () => (type === "workspace" ? workspacePermissions : api ? apiPermissions(api.id) : {}),
    [type, api?.id],
  );

  const filteredPermissionList = useMemo(() => {
    return filterPermissionList(permissionList, searchValue);
  }, [permissionList, searchValue]);

  // Filter out empty categories and actions
  const nonEmptyPermissionList = useMemo(() => {
    return Object.entries(filteredPermissionList).reduce<PermissionList>((acc, [category, permissions]) => {
      // Filter out actions that have no permissions
      const nonEmptyPermissions = Object.entries(permissions).reduce<PermissionCategory>((permAcc, [action, permissionData]) => {
        if (permissionData && permissionData.permission) {
          permAcc[action] = permissionData;
        }
        return permAcc;
      }, {});

      // Only include categories that have at least one permission
      if (Object.keys(nonEmptyPermissions).length > 0) {
        acc[category] = nonEmptyPermissions;
      }
      return acc;
    }, {});
  }, [filteredPermissionList]);

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
    selectedPermissions: selected.length > 0 ? initialState.selectedPermissions : [],
    categoryChecked: selected.length > 0 ? initialState.categoryChecked : {},
    rootChecked: selected.length > 0 ? initialState.rootChecked : false,
  });

  // Notify parent when selectedPermissions changes
  useEffect(() => {
    onPermissionChange(state.selectedPermissions);

    // eslint-disable-next-line react-hooks/exhaustive-deps
  }, [state.selectedPermissions]);

  const handleRootChecked = useCallback(() => {
    dispatch({ type: "TOGGLE_ROOT", permissionList });
  }, [permissionList]);

  const handleCategoryChecked = useCallback(
    (category: string) => {
      dispatch({ type: "TOGGLE_CATEGORY", category, permissionList });
    },
    [permissionList],
  );


  const handlePermissionChecked = useCallback(
    (permission: UnkeyPermission) => {
      dispatch({ type: "TOGGLE_PERMISSION", permission, permissionList });
    },
    [permissionList],
  );

  // if (Object.keys(nonEmptyPermissionList).length === 0 && searchValue) {
  //   return (
  //     <div className="flex flex-col gap-2">
  //       <p className="text-sm text-gray-10 ml-6 py-auto mt-1.5">No results found</p>
  //     </div>
  //   );
  // }

  // Don't render anything if there are no permissions at all
  if (Object.keys(nonEmptyPermissionList).length === 0) {
    setEmpty(true);
    return null;
  }
  else {
    setEmpty(false);
  }

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
            {Object.entries(nonEmptyPermissionList).map(([category, allPermissions]) => {
              return (
                <div
                  key={`${type === "workspace" ? "workspace" : api?.id}-${category}`}
                  className="flex flex-col gap-2 my-0 py-0 border-l border-grayA-5 ml-[31px]"
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
                        <div className="flex flex-col gap-2 my-0 py-0 border-l border-grayA-5 ml-[31px]">
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
              );
            })}
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
};
