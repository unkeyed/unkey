"use client";
import type { MenuItem } from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover";
import { KeysTableActionPopover } from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { Clone, PenWriting3, Trash } from "@unkey/icons";
import { useCallback } from "react";
import { DeleteRole } from "./components/delete-role";
import { EditRole } from "./components/edit-role";

type RolesTableActionsProps = {
  role: RoleBasic;
};

// `MAX_ATTACH_LIMIT` threshold for role attachments. Beyond this limit:
// - Role editing is disabled to prevent UI performance degradation
// - Warning callouts are shown to inform users of potential slowdowns
// - Prefetching of connected keys/permissions is skipped to reduce API load
export const MAX_ATTACH_LIMIT = 50;
export const RolesTableActions = ({ role }: RolesTableActionsProps) => {
  const trpcUtils = trpc.useUtils();

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

  // Those two `connectedKeys` and `connectedPerms` are already fetched  for table view so it's cached.
  const keysPreview = trpcUtils.authorization.roles.connectedKeys.getData({
    roleId: role.roleId,
    limit: 3,
  });
  const permsPreview = trpcUtils.authorization.roles.connectedPerms.getData({
    roleId: role.roleId,
    limit: 3,
  });

  const totalKeys = keysPreview?.totalCount || keysPreview?.items?.length || 0;
  const totalPerms = permsPreview?.totalCount || permsPreview?.items?.length || 0;

  const shouldPrefetch = totalKeys <= MAX_ATTACH_LIMIT && totalPerms <= MAX_ATTACH_LIMIT;

  const getRolesTableActionItems = (role: RoleBasic): MenuItem[] => {
    return [
      {
        id: "edit-role",
        label: "Edit role...",
        icon: <PenWriting3 size="md-regular" />,
        ActionComponent: (props) => (
          <EditRole role={role} {...props} shouldFetch={shouldPrefetch} />
        ),
        prefetch: async () => {
          shouldPrefetch
            ? await trpcUtils.authorization.roles.connectedKeysAndPerms.prefetch({
                roleId: role.roleId,
              })
            : undefined;
        },
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
