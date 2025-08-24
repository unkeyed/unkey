import { SelectedItemsList } from "@/components/selected-item-list";
import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";
import { CaretRight, Key2 } from "@unkey/icons";
import type { UnkeyPermission } from "@unkey/rbac";
import { Badge } from "@unkey/ui";
import { useMemo } from "react";
import { ROOT_KEY_CONSTANTS } from "../constants";
import { apiPermissions, workspacePermissions } from "../permissions";

type Props = {
  apiId: string;
  name: string;
  selectedPermissions: UnkeyPermission[];
  expandCount: number;
  title: string;
  removePermission: (permission: UnkeyPermission) => void;
};

type PermissionInfo = { permission: UnkeyPermission; category: string; action: string }[];

const PermissionBadgeList = ({
  apiId,
  name,
  selectedPermissions,
  title,
  expandCount,
  removePermission,
}: Props) => {
  // Flatten allPermissions into an array of {permission, action} objects
  const allPermissionsArray = useMemo(() => {
    const allPermissions =
      apiId === ROOT_KEY_CONSTANTS.WORKSPACE ? workspacePermissions : apiPermissions(apiId);
    return Object.entries(allPermissions).flatMap(([category, permissions]) =>
      Object.entries(permissions).map(([action, permissionData]) => ({
        permission: permissionData.permission,
        category,
        action,
      })),
    );
  }, [apiId]);

  const info = useMemo(
    () => findPermission(allPermissionsArray, selectedPermissions),
    [allPermissionsArray, selectedPermissions],
  );
  if (info.length === 0) {
    return null;
  }

  return info.length > expandCount ? (
    <div className="flex flex-col gap-2">
      <CollapsibleList
        info={info}
        title={title}
        expandCount={expandCount}
        removePermission={removePermission}
        name={name}
      />
    </div>
  ) : (
    <div className="flex flex-col gap-2">
      <ListTitle title={title} count={info.length} category={name} />
      <ListBadges info={info} removePermission={removePermission} />
    </div>
  );
};

const ListBadges = ({
  info,
  removePermission,
}: { info: PermissionInfo; removePermission: (permission: UnkeyPermission) => void }) => {
  // Stop propagation to prevent triggering parent collapsible when removing permissions
  const handleRemovePermissionClick = (id: string) => {
    const permission = info.find((p) => p.permission === id);
    if (permission) {
      removePermission(permission.permission);
    }
  };
  return (
    <SelectedItemsList
      className="pt-2 overflow-hidden"
      items={info.map((permission) => ({
        id: permission.permission,
        name: permission.action,
        description: permission.category,
      }))}
      gridCols={2}
      onRemoveItem={handleRemovePermissionClick}
      renderIcon={() => <Key2 size="sm-regular" className="text-grayA-11" aria-hidden="true" />}
      enableTransitions
      renderPrimaryText={(permission) => permission.name}
      renderSecondaryText={(permission) => permission.id}
    />
  );
};

type CollapsibleListProps = {
  info: PermissionInfo;
  title: string;
  expandCount: number;
  removePermission: (permission: UnkeyPermission) => void;
  name: string;
} & React.ComponentProps<typeof CollapsibleTrigger>;

const CollapsibleList = ({
  info,
  title,
  expandCount,
  removePermission,
  name,
  className,
  ...props
}: CollapsibleListProps) => {
  return (
    <Collapsible>
      <CollapsibleTrigger
        {...props}
        className={cn(
          "flex flex-row gap-3 transition-all [&[data-state=open]>svg]:rotate-90 w-full",
          className,
        )}
      >
        <ListTitle title={title} count={info.length} category={name} />
        <CaretRight className="w-4 h-4 transition-transform duration-200 ml-auto text-grayA-7" />
      </CollapsibleTrigger>
      <CollapsibleContent>
        <ListBadges info={info} removePermission={removePermission} />
      </CollapsibleContent>
    </Collapsible>
  );
};

const ListTitle = ({
  title,
  count,
  category,
}: { title: string; count: number; category: string }) => {
  return (
    <span className="text-[13px] flex-1 text-grayA-10 text-left flex items-center">
      {title}
      <span className="font-normal text-grayA-12 ml-1 font-mono">{category}</span>
      <Badge
        variant="primary"
        size="sm"
        className="text-[11px] font-normal text-gray-11 rounded-full px-2 ml-1 py-1 h-[18px] min-w-[22px] border-[1px] border-grayA-3 "
      >
        {count}
      </Badge>
    </span>
  );
};

const findPermission = (
  allPermissions: PermissionInfo,
  selectedPermissions: UnkeyPermission[],
): PermissionInfo => {
  if (!selectedPermissions || !Array.isArray(selectedPermissions)) {
    return [];
  }
  return selectedPermissions
    .map((permission) => {
      return allPermissions.find((p) => p.permission === permission);
    })
    .filter((item): item is { permission: UnkeyPermission; category: string; action: string } =>
      Boolean(item),
    );
};

export { PermissionBadgeList };
