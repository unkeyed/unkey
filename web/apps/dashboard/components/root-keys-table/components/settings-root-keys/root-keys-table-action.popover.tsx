"use client";
import {
  IconArrowDottedRotateAnticlockwiseOutline18,
  IconPenWriting3Outline18,
  IconTrashOutline18,
} from "nucleo-ui-outline-18";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { DeleteRootKey } from "./delete-root-key";
import { RotateRootKey } from "./rotate-root-key";

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
      icon: <IconPenWriting3Outline18 className="size-4" />,
      onClick: () => {
        onEditKey?.(rootKey);
      },
    },
    {
      id: "rotate-root-key",
      label: "Rotate root key...",
      icon: <IconArrowDottedRotateAnticlockwiseOutline18 className="size-4" />,
      ActionComponent: (props) => <RotateRootKey {...props} rootKeyDetails={rootKey} />,
      divider: true,
    },
    {
      id: "delete-root-key",
      label: "Delete root key",
      icon: <IconTrashOutline18 className="size-4" />,
      ActionComponent: (props) => <DeleteRootKey {...props} rootKeyDetails={rootKey} />,
    },
  ];
};
