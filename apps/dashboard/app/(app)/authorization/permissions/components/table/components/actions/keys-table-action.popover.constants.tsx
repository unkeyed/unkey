"use client";
import type { MenuItem } from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover";
import { KeysTableActionPopover } from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover";
import { toast } from "@/components/ui/toaster";
import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { Clone, PenWriting3 } from "@unkey/icons";
import { EditPermission } from "./components/edit-permission";

type PermissionsTableActionsProps = {
  permission: Permission;
};

export const PermissionsTableActions = ({ permission }: PermissionsTableActionsProps) => {
  const getPermissionsTableActionItems = (permission: Permission): MenuItem[] => {
    return [
      {
        id: "edit-permission",
        label: "Edit permission...",
        icon: <PenWriting3 size="md-regular" />,
        ActionComponent: (props) => <EditPermission permission={permission} {...props} />,
      },
      {
        id: "copy",
        label: "Copy permission",
        className: "mt-1",
        icon: <Clone size="md-regular" />,
        onClick: () => {
          navigator.clipboard
            .writeText(JSON.stringify(permission))
            .then(() => {
              toast.success("Role data copied to clipboard");
            })
            .catch((error) => {
              console.error("Failed to copy to clipboard:", error);
              toast.error("Failed to copy to clipboard");
            });
        },
        divider: true,
      },
      // {
      //   id: "delete-permision",
      //   label: "Delete permission",
      //   icon: <Trash size="md-regular" />,
      //   ActionComponent: (props) => (
      //     <DeleteRole {...props} roleDetails={permission} />
      //   ),
      // },
    ];
  };

  const menuItems = getPermissionsTableActionItems(permission);

  return <KeysTableActionPopover items={menuItems} />;
};
