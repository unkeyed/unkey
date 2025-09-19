"use client";
import {
  type MenuItem,
  TableActionPopover,
} from "@/components/logs/table-action.popover";
import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { Clone, PenWriting3, Trash } from "@unkey/icons";
import { toast } from "@unkey/ui";
import { DeletePermission } from "./components/delete-permission";
import { EditPermission } from "./components/edit-permission";

type PermissionsTableActionsProps = {
  permission: Permission;
};

export const PermissionsTableActions = ({
  permission,
}: PermissionsTableActionsProps) => {
  const getPermissionsTableActionItems = (
    permission: Permission
  ): MenuItem[] => {
    return [
      {
        id: "edit-permission",
        label: "Edit permission...",
        icon: <PenWriting3 iconsize="md-medium" />,
        ActionComponent: (props) => (
          <EditPermission permission={permission} {...props} />
        ),
      },
      {
        id: "copy",
        label: "Copy permission",
        className: "mt-1",
        icon: <Clone iconsize="md-medium" />,
        onClick: () => {
          navigator.clipboard
            .writeText(JSON.stringify(permission))
            .then(() => {
              toast.success("Permission data copied to clipboard");
            })
            .catch((error) => {
              console.error("Failed to copy to clipboard:", error);
              toast.error("Failed to copy to clipboard");
            });
        },
        divider: true,
      },
      {
        id: "delete-permision",
        label: "Delete permission",
        icon: <Trash iconsize="md-medium" />,
        ActionComponent: (props) => (
          <DeletePermission {...props} permissionDetails={permission} />
        ),
      },
    ];
  };

  const menuItems = getPermissionsTableActionItems(permission);

  return <TableActionPopover items={menuItems} />;
};
