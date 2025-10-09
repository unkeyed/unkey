"use client";
import {
  type MenuItem,
  TableActionPopover,
} from "@/components/logs/table-action.popover";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { PenWriting3, Trash } from "@unkey/icons";
import { DeleteRootKey } from "./components/delete-root-key";

type RootKeysTableActionsProps = {
  rootKey: RootKey;
  onEditKey?: (rootKey: RootKey) => void;
};

export const RootKeysTableActions = ({
  rootKey,
  onEditKey,
}: RootKeysTableActionsProps) => {
  const menuItems = getRootKeyTableActionItems(rootKey, onEditKey);
  return <TableActionPopover items={menuItems} />;
};

const getRootKeyTableActionItems = (
  rootKey: RootKey,
  onEditKey?: (rootKey: RootKey) => void
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
      id: "delete-root-key",
      label: "Delete root key",
      icon: <Trash iconSize="md-medium" />,
      ActionComponent: (props) => (
        <DeleteRootKey {...props} rootKeyDetails={rootKey} />
      ),
    },
  ];
};
