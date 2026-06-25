"use client";

import { Collapsible, CollapsibleContent } from "@/components/ui/collapsible";
import { match } from "@unkey/match";
import type { UnkeyPermission } from "@unkey/rbac";
import { useCallback, useEffect, useState } from "react";
import { ROOT_KEY_MESSAGES } from "../constants";
import { usePermissions } from "../hooks/use-permissions";
import type { PermissionScope } from "../permissions";
import { ExpandableCategory } from "./expandable-category";
import { HighlightedText } from "./highlighted-text";
import { PermissionToggle } from "./permission-toggle";

type PermissionContentListProps = {
  scope: PermissionScope;
  onPermissionChange: (permissions: UnkeyPermission[]) => void;
  selected: UnkeyPermission[];
  searchValue?: string;
};

type ScopeHeader = {
  name: string;
  description: string;
  key: string;
};

const getScopeHeader = (scope: PermissionScope): ScopeHeader =>
  match(scope)
    .with({ kind: "workspace" }, () => ({
      name: "Workspace",
      description: ROOT_KEY_MESSAGES.DESCRIPTIONS.WORKSPACE,
      key: "workspace",
    }))
    .with({ kind: "api" }, ({ name, id }) => ({
      name,
      description: `${ROOT_KEY_MESSAGES.DESCRIPTIONS.API} ${name}`,
      key: id,
    }))
    .with({ kind: "project" }, ({ name, id }) => ({
      name,
      description: `${ROOT_KEY_MESSAGES.DESCRIPTIONS.PROJECT} ${name}`,
      key: id,
    }))
    .with({ kind: "app" }, ({ name, id }) => ({
      name,
      description: `${ROOT_KEY_MESSAGES.DESCRIPTIONS.APP} ${name}`,
      key: id,
    }))
    .with({ kind: "environment" }, ({ name, id }) => ({
      name,
      description: `${ROOT_KEY_MESSAGES.DESCRIPTIONS.ENVIRONMENT} ${name}`,
      key: id,
    }))
    .exhaustive();

export const PermissionContentList = ({
  scope,
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
    scope,
    selected,
    searchValue,
    onPermissionChange,
  });

  const [expandedCategories, setExpandedCategories] = useState<Set<string>>(new Set());
  const [isRootExpanded, setIsRootExpanded] = useState(false);

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

  if (totalPermissions === 0) {
    return null;
  }

  const header = getScopeHeader(scope);

  return (
    <div className="flex flex-col w-full grow-0 max-w-[380px] px-2">
      <Collapsible
        open={isRootExpanded}
        onOpenChange={setIsRootExpanded}
        className="hover:bg-grayA-3 rounded-lg mb-2"
      >
        <ExpandableCategory
          category={header.name}
          description={header.description}
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
                  key={`${header.key}-${category}`}
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
