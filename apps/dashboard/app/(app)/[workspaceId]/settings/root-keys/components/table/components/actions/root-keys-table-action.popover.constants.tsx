"use client";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { PenWriting3, Trash } from "@unkey/icons";
import type { AppRouterInstance } from "next/dist/shared/lib/app-router-context.shared-runtime";
import { useRouter } from "next/navigation";
import { DeleteRootKey } from "./components/delete-root-key";

type RootKeysTableActionsProps = {
  rootKey: RootKey;
  workspaceId: string;
};

export const RootKeysTableActions = ({ rootKey, workspaceId }: RootKeysTableActionsProps) => {
  const router = useRouter();
  const menuItems = getRootKeyTableActionItems(rootKey, router, workspaceId);
  return <TableActionPopover items={menuItems} />;
};

const getRootKeyTableActionItems = (
  rootKey: RootKey,
  router: AppRouterInstance,
  workspaceId: string,
): MenuItem[] => {
  return [
    {
      id: "edit-root-key",
      label: "Edit root key...",
      icon: <PenWriting3 size="md-regular" />,
      onClick: () => {
        router.push(`/${workspaceId}/settings/root-keys/${rootKey.id}`);
      },
      divider: true,
    },
    {
      id: "delete-root-key",
      label: "Delete root key",
      icon: <Trash size="md-regular" />,
      ActionComponent: (props) => <DeleteRootKey {...props} rootKeyDetails={rootKey} />,
    },
  ];
};
