"use client";

import {
  type MenuItem,
  TableActionPopoverDefaultTrigger,
} from "@/components/logs/table-action.popover";
import { trpc } from "@/lib/trpc/client";
import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { Clone, PenWriting3, Trash } from "@unkey/icons";
import { toast } from "@unkey/ui";
import dynamic from "next/dynamic";
import { MAX_KEYS_FETCH_LIMIT } from "../../../upsert-role/components/assign-key/hooks/use-fetch-keys";
import { MAX_PERMS_FETCH_LIMIT } from "../../../upsert-role/components/assign-permission/hooks/use-fetch-permissions";
import { DeleteRole } from "./components/delete-role";
import { EditRole } from "./components/edit-role";

// Wrapper component to handle React Loadable props
const LoadingTrigger = () => <TableActionPopoverDefaultTrigger />;

const KeysTableActionPopover = dynamic(
  () =>
    import("@/components/logs/table-action.popover").then(
      (mod) => mod.TableActionPopover
    ),
  {
    loading: LoadingTrigger,
  }
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
  trpcUtils: ReturnType<typeof trpc.useUtils>
): MenuItem[] => {
  return [
    {
      id: "edit-role",
      label: "Edit role...",
      icon: <PenWriting3 iconSize="md-medium" />,
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
      icon: <Clone iconSize="md-medium" />,
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
      icon: <Trash iconSize="md-medium" />,
      ActionComponent: (props) => <DeleteRole {...props} roleDetails={role} />,
    },
  ];
};
