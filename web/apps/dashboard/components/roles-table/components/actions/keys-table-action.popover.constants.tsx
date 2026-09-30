"use client";

import { MAX_KEYS_FETCH_LIMIT } from "@/app/(app)/[workspaceSlug]/authorization/roles/components/upsert-role/components/assign-key/hooks/use-fetch-keys";
import {
  type MenuItem,
  TableActionPopoverDefaultTrigger,
} from "@/components/logs/table-action.popover";
import { permissionsQueryOptions } from "@/hooks/use-fetch-permissions";
import { trpc } from "@/lib/trpc/client";
import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { type QueryClient, useQueryClient } from "@tanstack/react-query";
import { IconCloneOutline18, IconPenWriting3Outline18, IconTrashOutline18 } from "@unkey/icons";
import { toast } from "@unkey/ui";
import dynamic from "next/dynamic";
import { DeleteRole } from "./components/delete-role";
import { EditRole } from "./components/edit-role";

// Wrapper component to handle React Loadable props
const LoadingTrigger = () => <TableActionPopoverDefaultTrigger />;

const KeysTableActionPopover = dynamic(
  () => import("@/components/logs/table-action.popover").then((mod) => mod.TableActionPopover),
  {
    loading: LoadingTrigger,
  },
);

type RolesTableActionsProps = {
  role: RoleBasic;
};

export const RolesTableActions = ({ role }: RolesTableActionsProps) => {
  const trpcUtils = trpc.useUtils();
  const queryClient = useQueryClient();
  const menuItems = getRolesTableActionItems(role, trpcUtils, queryClient);

  return <KeysTableActionPopover items={menuItems} />;
};

const getRolesTableActionItems = (
  role: RoleBasic,
  trpcUtils: ReturnType<typeof trpc.useUtils>,
  queryClient: QueryClient,
): MenuItem[] => {
  return [
    {
      id: "edit-role",
      label: "Edit role...",
      icon: <IconPenWriting3Outline18 className="size-3.5" />,
      ActionComponent: (props) => <EditRole role={role} {...props} />,
      prefetch: async () => {
        await Promise.all([
          trpcUtils.authorization.roles.keys.query.prefetchInfinite({
            limit: MAX_KEYS_FETCH_LIMIT,
          }),
          queryClient.prefetchInfiniteQuery(permissionsQueryOptions()),
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
      icon: <IconCloneOutline18 className="size-3.5" />,
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
      icon: <IconTrashOutline18 className="size-3.5" />,
      ActionComponent: (props) => <DeleteRole {...props} roleDetails={role} />,
    },
  ];
};
