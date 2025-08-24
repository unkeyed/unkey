"use client";

import { Collapsible, CollapsibleContent } from "@/components/ui/collapsible";
import type { UnkeyPermission } from "@unkey/rbac";
import { useCallback, useEffect, useState } from "react";
import { ROOT_KEY_MESSAGES } from "../constants";
import { usePermissions } from "../hooks/use-permissions";
import { ExpandableCategory } from "./expandable-category";
import { HighlightedText } from "./highlighted-text";
import { PermissionToggle } from "./permission-toggle";

type PermissionContentListProps = {
  type: "workspace" | "api";
  api?: { id: string; name: string };
  onPermissionChange: (permissions: UnkeyPermission[]) => void;
  selected: UnkeyPermission[];
  searchValue?: string;
};

export const PermissionContentList = ({
  type,
  api,
  onPermissionChange,
  selected,
  searchValue,
}: PermissionContentListProps) => {
  const {
    state,
    filteredPermissionList,
    handleRootToggle,
    handleCategoryToggle,
    handlePermissionToggle,
  } = usePermissions({
    type,
    api,
    selected,
    searchValue,
    onPermissionChange,
  });

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

  const totalPermissions = Object.keys(filteredPermissionList).reduce(
    (acc, category) => acc + Object.keys(filteredPermissionList[category]).length,
    0,
  );

  if (totalPermissions === 0) {
    return null;
  }

  const handleCategoryToggleExpanded = useCallback((category: string, open: boolean) => {
    setExpandedCategories((prev) => {
      const newSet = new Set(prev);
      if (open) {
        newSet.add(category);
      } else {
        newSet.delete(category);
      }
      return newSet;
    });
  }, []);
  return (
    <div className="flex flex-col w-full grow-0 max-w-[380px] px-2">
      <Collapsible
        open={isRootExpanded}
        onOpenChange={setIsRootExpanded}
        className="hover:bg-grayA-3 rounded-lg mb-2"
      >
        <ExpandableCategory
          category={type === "workspace" ? "Workspace" : (api?.name ?? "API")}
          description={
            type === "workspace"
              ? ROOT_KEY_MESSAGES.DESCRIPTIONS.WORKSPACE
              : `${ROOT_KEY_MESSAGES.DESCRIPTIONS.API} ${api?.name ?? "API"}`
          }
          checked={state.rootChecked}
          setChecked={handleRootToggle}
          count={totalPermissions}
        />

        <CollapsibleContent>
          <div className="flex">
            <div className="flex flex-col min-h-full border-r border-grayA-5 mb-2 ml-5" />
            <div className="flex flex-col h-full ml-2 w-full min-w-0">
              {Object.entries(filteredPermissionList).map(([category, allPermissions]) => (
                <Collapsible
                  key={`${type === "workspace" ? "workspace" : api?.id}-${category}`}
                  className="rounded-lg hover:bg-grayA-3 p-0 m-0 w-full min-w-0"
                  open={expandedCategories.has(category)}
                  onOpenChange={(open) => handleCategoryToggleExpanded(category, open)}
                >
                  <div className="flex-1 justify-start items-start w-full min-w-0">
                    <ExpandableCategory
                      category={category}
                      checked={state.categoryChecked[category]}
                      setChecked={() => handleCategoryToggle(category)}
                      count={Object.keys(allPermissions).length}
                    />
                    <CollapsibleContent>
                      <div className="flex w-full">
                        <div className="flex-1 border-r border-grayA-5 max-h-full w-4 mb-2 ml-[20px]" />
                        <div className="flex flex-col min-w-0 mr-2 w-full justify-start items-start ">
                          {Object.entries(allPermissions).map(
                            ([action, { description, permission }]) => (
                              <PermissionToggle
                                key={action}
                                category={
                                  <HighlightedText text={category} searchValue={searchValue} />
                                }
                                className="pr-2"
                                label={<HighlightedText text={action} searchValue={searchValue} />}
                                description={description}
                                checked={state.selectedPermissions.includes(permission)}
                                setChecked={() => handlePermissionToggle(permission)}
                              />
                            ),
                          )}
                        </div>
                      </div>
                    </CollapsibleContent>
                  </div>
                </Collapsible>
              ))}
            </div>
          </div>
        </CollapsibleContent>
      </Collapsible>
    </div>
  );
};
