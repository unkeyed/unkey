"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { Clone, PenWriting3, Trash } from "@unkey/icons";
import { useMemo } from "react";

type EnvVarActionMenuProps = {
  variableKey: string;
};

export function EnvVarActionMenu({ variableKey }: EnvVarActionMenuProps) {
  const menuItems = useMemo((): MenuItem[] => {
    return [
      {
        id: "edit",
        label: "Edit",
        icon: <PenWriting3 iconSize="md-regular" />,
        onClick: (e) => {
          e.stopPropagation();
          // TODO: wire up edit action
        },
      },
      {
        id: "delete",
        label: "Delete",
        icon: <Trash iconSize="md-regular" />,
        divider: true,
        onClick: (e) => {
          e.stopPropagation();
          // TODO: wire up delete action
        },
      },
      {
        id: "copy",
        label: "Copy to Clipboard",
        icon: <Clone iconSize="md-regular" />,
        onClick: (e) => {
          e.stopPropagation();
          // TODO: wire up copy action
        },
      },
    ];
  }, []);

  return <TableActionPopover items={menuItems} />;
}
