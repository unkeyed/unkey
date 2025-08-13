import { Collapsible, CollapsibleContent, CollapsibleTrigger } from "@/components/ui/collapsible";
import { cn } from "@/lib/utils";
import { ChevronDown, XMark } from "@unkey/icons";
import type { UnkeyPermission } from "@unkey/rbac";
import { Badge, Button } from "@unkey/ui";
import { useMemo } from "react";
import { apiPermissions, workspacePermissions } from "../../../[keyId]/permissions/permissions";

type Props = {
  apiId: string;
  name: string;
  selectedPermissions: UnkeyPermission[];
  expandCount: number;
  title: string;
  removePermission: (permission: string) => void;
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
  const workspace = workspacePermissions;
  const allPermissions = apiId === "workspace" ? workspace : apiPermissions(apiId);

  // Flatten allPermissions into an array of {permission, action} objects
  const allPermissionsArray = useMemo(
    () =>
      Object.entries(allPermissions).flatMap(([category, permissions]) =>
        Object.entries(permissions).map(([action, permissionData]) => ({
          permission: permissionData.permission,
          category,
          action,
        })),
      ),
    [allPermissions],
  );

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
}: { info: PermissionInfo; removePermission: (permission: string) => void }) => {
  const handleRemovePermission = (e: React.MouseEvent<HTMLButtonElement>, permission: string) => {
    e.stopPropagation();
    removePermission(permission);
  };
  return (
    <div className="flex flex-wrap gap-2 pt-2">
      {info?.map((permission) => {
        if (!permission) {
          return null;
        }

        return (
          <Badge
            key={permission.permission}
            variant="primary"
            className="flex items-center h-[22px] p-[6px] px-2 text-xs font-normal text-grayA-11 hover:bg-grayA-2 hover:text-grayA-12 gap-2"
          >
            <span>{permission.action}</span>
            <Button
              variant="ghost"
              size="icon"
              className="w-4 h-4"
              onClick={(e) => handleRemovePermission(e, permission.permission)}
            >
              <XMark className="w-4 h-4" />
            </Button>
          </Badge>
        );
      })}
    </div>
  );
};

type CollapsibleListProps = {
  info: PermissionInfo;
  title: string;
  expandCount: number;
  removePermission: (permission: string) => void;
  name: string;
  className?: string;
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
          "flex flex-row gap-3 transition-all [&[data-state=open]>svg]:rotate-180 w-full",
          className,
        )}
      >
        <ListTitle title={title} count={info.length} category={name} />
        <ChevronDown className="w-4 h-4 transition-transform duration-200 ml-auto text-grayA-8" />
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
    <p className="text-sm w-full text-grayA-10 justify-start text-left">
      {title}
      <span className="font-bold text-gray-11 ml-2">{category}</span>
      <Badge
        variant="primary"
        size="sm"
        className="text-[11px] font-normal text-grayA-11 rounded-full px-2 ml-4 h-[18px] min-w-[22px] border-[1px] border-grayA-3 "
      >
        {count}
      </Badge>
    </p>
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
