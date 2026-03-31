"use client";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { PenWriting3, Refresh3, Trash } from "@unkey/icons";
import { DeleteRootKey } from "./delete-root-key";
import { RerollRootKey } from "./reroll-root-key";

type RootKeysTableActionsProps = {
  rootKey: RootKey;
  onEditKey?: (rootKey: RootKey) => void;
};

export const RootKeysTableActions = ({ rootKey, onEditKey }: RootKeysTableActionsProps) => {
  const menuItems = getRootKeyTableActionItems(rootKey, onEditKey);
  return <TableActionPopover items={menuItems} />;
};

const getRootKeyTableActionItems = (
  rootKey: RootKey,
  onEditKey?: (rootKey: RootKey) => void,
): MenuItem[] => {
  return [
    {
      id: "edit-root-key",
      label: "Edit root key...",
      icon: <PenWriting3 iconSize="md-medium" />,
      onClick: () => {
        onEditKey?.(rootKey);
      },
      divider: true,
    },
    {
      id: "reroll-root-key",
      label: "Reroll root key...",
      icon: <Refresh3 iconSize="md-medium" />,
      ActionComponent: (props) => <RerollRootKey {...props} rootKeyDetails={rootKey} />,
      divider: true,
    },
    {
      id: "delete-root-key",
      label: "Delete root key",
      icon: <Trash iconSize="md-medium" />,
      ActionComponent: (props) => <DeleteRootKey {...props} rootKeyDetails={rootKey} />,
    },
  ];
};
