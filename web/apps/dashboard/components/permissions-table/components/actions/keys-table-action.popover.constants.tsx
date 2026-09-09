"use client";
import { toast } from "@unkey/ui";
import {
  IconCloneOutline18,
  IconPenWriting3Outline18,
  IconTrashOutline18,
} from "nucleo-ui-outline-18";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { DeletePermission } from "./components/delete-permission";
import { EditPermission } from "./components/edit-permission";

export const getPermissionsTableActionItems = (permission: Permission): MenuItem[] => [
  {
    id: "edit-permission",
    label: "Edit permission...",
    icon: <IconPenWriting3Outline18 className="size-4" />,
    ActionComponent: (props) => <EditPermission permission={permission} {...props} />,
  },
  {
    id: "copy",
    label: "Copy permission",
    className: "mt-1",
    icon: <IconCloneOutline18 className="size-4" />,
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
    id: "delete-permission",
    label: "Delete permission",
    icon: <IconTrashOutline18 className="size-4" />,
    ActionComponent: (props) => <DeletePermission {...props} permissionDetails={permission} />,
  },
];

type PermissionsTableActionsProps = {
  permission: Permission;
};

export const PermissionsTableActions = ({ permission }: PermissionsTableActionsProps) => {
  return <TableActionPopover items={getPermissionsTableActionItems(permission)} />;
};
