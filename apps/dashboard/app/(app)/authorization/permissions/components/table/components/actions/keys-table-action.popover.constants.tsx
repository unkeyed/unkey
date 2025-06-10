"use client";
import type { MenuItem } from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover";
import { KeysTableActionPopover } from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
import { Clone, PenWriting3, Trash } from "@unkey/icons";
import { DeleteRole } from "./components/delete-role";
import { EditRole } from "./components/edit-role";

type PermissionsTableActionsProps = {
  permission: Permission;
};

export const PermissionsTableActions = ({ permission }: PermissionsTableActionsProps) => {
  const trpcUtils = trpc.useUtils();

  const getPermissionsTableActionItems = (permission: Permission): MenuItem[] => {
    return [
      {
        id: "edit-permission",
        label: "Edit permission...",
        icon: <PenWriting3 size="md-regular" />,
        ActionComponent: (props) => <EditRole role={permission} {...props} />,
        prefetch: async () => {
          await trpcUtils.authorization.roles.connectedKeysAndPerms.prefetch({
            roleId: permission.roleId,
          });
        },
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
      {
        id: "delete-role",
        label: "Delete role",
        icon: <Trash size="md-regular" />,
        ActionComponent: (props) => <DeleteRole {...props} roleDetails={permission} />,
      },
    ];
  };

  const menuItems = getPermissionsTableActionItems(permission);

  return <KeysTableActionPopover items={menuItems} />;
};
