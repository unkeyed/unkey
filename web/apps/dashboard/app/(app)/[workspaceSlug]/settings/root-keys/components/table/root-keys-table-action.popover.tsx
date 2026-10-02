"use client";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import {
  IconArrowDottedRotateAnticlockwiseOutline18,
  IconPenWriting3Outline18,
  IconTrashOutline18,
} from "@unkey/icons";
import { DeleteRootKey, DeleteRootKeyV2 } from "./delete-root-key";
import { RotateRootKey, RotateRootKeyV2 } from "./rotate-root-key";

type RootKeysTableActionsProps = {
  rootKey: RootKey;
  onEditKey?: (rootKey: RootKey) => void;
  transport: "legacy" | "v2";
};

export const RootKeysTableActions = ({
  rootKey,
  onEditKey,
  transport,
}: RootKeysTableActionsProps) => {
  const menuItems = getRootKeyTableActionItems(rootKey, onEditKey, transport);
  return <TableActionPopover items={menuItems} />;
};

const getRootKeyTableActionItems = (
  rootKey: RootKey,
  onEditKey?: (rootKey: RootKey) => void,
  transport: "legacy" | "v2" = "legacy",
): MenuItem[] => {
  return [
    {
      id: "edit-root-key",
      label: "Edit root key...",
      icon: <IconPenWriting3Outline18 className="size-3.5" />,
      onClick: () => {
        onEditKey?.(rootKey);
      },
    },
    {
      id: "rotate-root-key",
      label: "Rotate root key...",
      icon: <IconArrowDottedRotateAnticlockwiseOutline18 className="size-3.5" />,
      ActionComponent: (props) =>
        transport === "v2" ? (
          <RotateRootKeyV2 {...props} rootKeyDetails={rootKey} />
        ) : (
          <RotateRootKey {...props} rootKeyDetails={rootKey} />
        ),
      divider: true,
    },
    {
      id: "delete-root-key",
      label: "Delete root key",
      icon: <IconTrashOutline18 className="size-3.5" />,
      ActionComponent: (props) =>
        transport === "v2" ? (
          <DeleteRootKeyV2 {...props} rootKeyDetails={rootKey} />
        ) : (
          <DeleteRootKey {...props} rootKeyDetails={rootKey} />
        ),
    },
  ];
};
