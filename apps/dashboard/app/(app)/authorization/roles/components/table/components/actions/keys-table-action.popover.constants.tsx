"use client";
import {
  KeysTableActionPopoverDefaultTrigger,
  type MenuItem,
} from "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover";
import { toast } from "@/components/ui/toaster";
import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { Clone, PenWriting3, Trash } from "@unkey/icons";
import dynamic from "next/dynamic";
import { MAX_KEYS_FETCH_LIMIT } from "../../../upsert-role/components/assign-key/hooks/use-fetch-keys";
import { MAX_PERMS_FETCH_LIMIT } from "../../../upsert-role/components/assign-permission/hooks/use-fetch-permissions";
import { DeleteRole } from "./components/delete-role";
import { EditRole } from "./components/edit-role";
import { trpc } from "@/lib/trpc/client";

const KeysTableActionPopover = dynamic(
  () =>
    import(
      "@/app/(app)/apis/[apiId]/keys/[keyAuthId]/_components/components/table/components/actions/keys-table-action.popover"
    ).then((mod) => mod.KeysTableActionPopover),
  {
    loading: KeysTableActionPopoverDefaultTrigger,
  },
);

type RolesTableActionsProps = {
  role: RoleBasic;
};

export const RolesTableActions = ({ role }: RolesTableActionsProps) => {
  const trpcUtils = trpc.useUtils();
  const menuItems = getRolesTableActionItems(role, trpcUtils);

  return <KeysTableActionPopover items={menuItems} />;
};

const getRolesTableActionItems = (
  role: RoleBasic,
  trpcUtils: ReturnType<typeof trpc.useUtils>,
): MenuItem[] => {
  return [
    {
      id: "edit-role",
      label: "Edit role...",
      icon: <PenWriting3 size="md-regular" />,
      ActionComponent: (props) => <EditRole role={role} {...props} />,
      prefetch: async () => {
        await Promise.all([
          trpcUtils.authorization.roles.keys.query.prefetchInfinite({
            limit: MAX_KEYS_FETCH_LIMIT,
          }),
          trpcUtils.authorization.roles.permissions.query.prefetchInfinite({
            limit: MAX_PERMS_FETCH_LIMIT,
          }),
          trpcUtils.authorization.roles.connectedKeysAndPerms.prefetch({
            roleId: role.roleId,
          }),
        ]);
      },
    },
    {
      id: "copy",
      label: "Copy role",
      className: "mt-1",
      icon: <Clone size="md-regular" />,
      onClick: () => {
        navigator.clipboard
          .writeText(JSON.stringify(role))
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
      ActionComponent: (props) => <DeleteRole {...props} roleDetails={role} />,
    },
  ];
};
