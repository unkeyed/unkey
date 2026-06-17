"use client";

import { SelectedItemsList } from "@/components/selected-item-list";
import { Key2 } from "@unkey/icons";
import type { UnkeyPermission } from "@unkey/rbac";

type Props = {
  selectedPermissions: UnkeyPermission[];
  removePermission: (permission: UnkeyPermission) => void;
};

function describePermission(permission: UnkeyPermission): { name: string; description: string } {
  const value = String(permission);
  const [resource, action] = value.split("#");

  if (!resource || !action) {
    return { name: value, description: "Legacy permission" };
  }

  const path = resource.split(":").at(3) ?? resource;
  return {
    name: action === "*" ? "All actions" : action,
    description: path,
  };
}

export function UrnPermissionBadgeList({ selectedPermissions, removePermission }: Props) {
  if (selectedPermissions.length === 0) {
    return null;
  }

  const items = selectedPermissions.map((permission) => ({
    id: permission,
    ...describePermission(permission),
  }));

  return (
    <div className="flex flex-col gap-2">
      <span className="text-[13px] text-grayA-10 text-left">
        Selected permissions
        <span className="font-mono text-grayA-12 ml-1">{selectedPermissions.length}</span>
      </span>
      <SelectedItemsList
        className="pt-2 overflow-hidden"
        items={items}
        gridCols={1}
        onRemoveItem={(id) => removePermission(id as UnkeyPermission)}
        renderIcon={() => (
          <Key2 iconSize="sm-regular" className="text-grayA-11" aria-hidden="true" />
        )}
        enableTransitions
        renderPrimaryText={(permission) => permission.name}
        renderSecondaryText={(permission) => permission.description}
      />
    </div>
  );
}
