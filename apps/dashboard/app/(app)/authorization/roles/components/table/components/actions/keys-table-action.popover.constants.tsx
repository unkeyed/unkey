"use client";
import type { MenuItem } from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover";
import { KeysTableActionPopover } from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover";
import { toast } from "@/components/ui/toaster";
import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { Clone, PenWriting3, Trash } from "@unkey/icons";
import { useCallback } from "react";
import { useRoleLimits } from "../../hooks/use-role-limits";
import { DeleteRole } from "./components/delete-role";
import { EditRole } from "./components/edit-role";

type RolesTableActionsProps = {
  role: RoleBasic;
};

export const RolesTableActions = ({ role }: RolesTableActionsProps) => {
  const { prefetchIfAllowed } = useRoleLimits(role.roleId);

  const handleCopy = useCallback(() => {
    navigator.clipboard
      .writeText(JSON.stringify(role))
      .then(() => {
        toast.success("Role data copied to clipboard");
      })
      .catch((error) => {
        console.error("Failed to copy to clipboard:", error);
        toast.error("Failed to copy to clipboard");
      });
  }, [role]);

  const getRolesTableActionItems = (role: RoleBasic): MenuItem[] => {
    return [
      {
        id: "edit-role",
        label: "Edit role...",
        icon: <PenWriting3 size="md-regular" />,
        ActionComponent: (props) => <EditRole role={role} {...props} />,
        prefetch: prefetchIfAllowed,
      },
      {
        id: "copy",
        label: "Copy role",
        className: "mt-1",
        icon: <Clone size="md-regular" />,
        onClick: handleCopy,
        divider: true,
      },
      {
        id: "delete-role",
        label: "Delete role",
        icon: <Trash size="md-regular" />,
        ActionComponent: (props) => <DeleteRole {...props} roleDetails={role} />,
      },
    ];
  };

  const menuItems = getRolesTableActionItems(role);
  return <KeysTableActionPopover items={menuItems} />;
};
