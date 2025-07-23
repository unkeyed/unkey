"use client";
import { Collapsible, CollapsibleContent } from "@/components/ui/collapsible";
import type { CheckedState } from "@radix-ui/react-checkbox";
import type { UnkeyPermission } from "@unkey/rbac";
import { useEffect, useState } from "react";
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
};

export const PermissionContentList = ({ type, api }: Props) => {
  const [selectedPermissions, setSelectedPermissions] = useState<string[]>([]);
  const permissionList =
    type === "workspace" ? workspacePermissions : api ? apiPermissions(api.id) : {};
  const [rootChecked, setRootChecked] = useState<CheckedState>(false);
  const [categoryChecked, setCategoryChecked] = useState<Record<string, CheckedState>>({});

  useEffect(() => {
    if (selectedPermissions.length === 0) {
      setRootChecked(false);
    } else if (
      selectedPermissions.length ===
      Object.values(permissionList).flatMap((category) =>
        Object.values(category).map(({ permission }) => permission),
      ).length
    ) {
      setRootChecked(true);
    } else {
      setRootChecked("indeterminate");
    }
    Object.entries(permissionList).forEach(([category, allPermissions]) => {
      const allPermissionNames = Object.values(allPermissions).map(({ permission }) => permission);
      if (allPermissionNames.every((p) => selectedPermissions.includes(p))) {
        setCategoryChecked((prev) => ({ ...prev, [category]: true }));
      } else if (allPermissionNames.some((p) => selectedPermissions.includes(p))) {
        setCategoryChecked((prev) => ({ ...prev, [category]: "indeterminate" }));
      } else {
        setCategoryChecked((prev) => ({ ...prev, [category]: false }));
      }
    });
  }, [selectedPermissions, rootChecked]);

  const handleRootChecked = () => {
    if (rootChecked === "indeterminate") {
      setRootChecked(false);
      setSelectedPermissions([]);
      return;
    }
    if (rootChecked === true) {
      setRootChecked(false);
      setSelectedPermissions([]);
      return;
    }
    if (rootChecked === false) {
      setRootChecked(true);
      setSelectedPermissions(
        Object.values(permissionList).flatMap((category) =>
          Object.values(category).map(({ permission }) => permission),
        ),
      );
      return;
    }
    const allPermissionNames = Object.values(permissionList).flatMap((category) =>
      Object.values(category).map(({ permission }) => permission),
    );
    if (allPermissionNames.every((permission) => selectedPermissions.includes(permission))) {
      setRootChecked(true);
    } else if (allPermissionNames.some((permission) => selectedPermissions.includes(permission))) {
      setRootChecked("indeterminate");
    } else {
      setRootChecked(false);
    }
  };

  const handleCategoryChecked = (category: string) => {
    if (categoryChecked[category] === "indeterminate") {
      setSelectedPermissions((prev) =>
        prev.filter(
          (p) =>
            !Object.values(permissionList[category as keyof typeof permissionList])
              .map(({ permission }) => permission)
              .includes(p as UnkeyPermission),
        ),
      );
    } else if (categoryChecked[category] === true) {
      // Remove all permissions in this category
      setSelectedPermissions((prev) =>
        prev.filter(
          (p) =>
            !Object.values(permissionList[category as keyof typeof permissionList])
              .map(({ permission }) => permission)
              .includes(p as UnkeyPermission),
        ),
      );
    } else {
      setSelectedPermissions((prev) => [
        ...prev,
        ...Object.values(permissionList[category as keyof typeof permissionList]).map(
          ({ permission }) => permission,
        ),
      ]);
    }
  };

  const handlePermissionChecked = (permission: string) => {
    if (selectedPermissions.includes(permission)) {
      setSelectedPermissions((prev) => prev.filter((p) => p !== permission));
    } else {
      setSelectedPermissions((prev) => [...prev, permission]);
    }
    // Update the category checked state
    // Find which category this permission belongs to
    Object.entries(permissionList).forEach(([category, allPermissions]) => {
      const isInCategory = Object.values(allPermissions)
        .map(({ permission }) => permission)
        .includes(permission as UnkeyPermission);
      if (!isInCategory) {
        return;
      }
      const allPermissionNames = Object.values(allPermissions).map(({ permission }) => permission);
      // If all permissions in the category are selected, set the category to true
      if (allPermissionNames.every((p) => selectedPermissions.includes(p))) {
        setCategoryChecked((prev) => ({ ...prev, [category]: true }));
      }
      // If some permissions in the category are selected, set the category to indeterminate
      else if (allPermissionNames.some((p) => selectedPermissions.includes(p))) {
        setCategoryChecked((prev) => ({ ...prev, [category]: "indeterminate" }));
      }
      // If no permissions in the category are selected, set the category to false
      else {
        setCategoryChecked((prev) => ({ ...prev, [category]: false }));
      }
    });
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
          checked={rootChecked}
          setChecked={() => handleRootChecked()}
        />
        <CollapsibleContent>
          <div className="flex flex-col">
            {Object.entries(permissionList).map(([category, allPermissions]) => {
              const allPermissionNames = Object.values(allPermissions).map(
                ({ permission }) => permission,
              );
              return (
                <>
                  <div
                    key={`${type === "workspace" ? "workspace" : api?.id}-${category}`}
                    className="flex flex-col gap-2 my-0 py-0 border-l ml-[23px]"
                  >
                    <div className="flex flex-col my-0 py-0">
                      <Collapsible>
                        <ExpandableCategory
                          category={category}
                          checked={categoryChecked[category]}
                          description={""}
                          setChecked={() => handleCategoryChecked(category)}
                        />
                        <CollapsibleContent>
                          <div className="flex flex-col gap-2 my-0 py-0 border-l ml-6">
                            {Object.entries(allPermissions).map(
                              ([action, { description, permission }]) => (
                                <PermissionToggle
                                  key={action}
                                  label={action}
                                  description={description}
                                  checked={selectedPermissions.includes(permission)}
                                  setChecked={(checked: boolean) =>
                                    handlePermissionChecked(permission)
                                  }
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
