"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { collection } from "@/lib/collections";
import { Dots, PenWriting3, Trash } from "@unkey/icons";
import { Button, ConfirmPopover } from "@unkey/ui";
import { useRef, useState } from "react";
import { EnvVarGroupRenameDialog } from "./env-var-group-rename-dialog";
import type { EnvVarItem } from "./env-var-item-row";

type EnvVarGroupActionMenuProps = {
  groupKey: string;
  items: EnvVarItem[];
};

export function EnvVarGroupActionMenu({ groupKey, items }: EnvVarGroupActionMenuProps) {
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const [isRenameOpen, setIsRenameOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);

  const environmentNames = items.map((i) => i.environmentName).join(", ");

  const menuItems: MenuItem[] = [
    {
      id: "rename",
      label: "Rename in all environments",
      icon: <PenWriting3 iconSize="md-regular" />,
      onClick: (e) => {
        e.stopPropagation();
        setIsRenameOpen(true);
      },
    },
    {
      id: "delete",
      label: "Delete from all environments",
      icon: <Trash iconSize="md-regular" />,
      divider: true,
      onClick: (e) => {
        e.stopPropagation();
        setIsDeleteConfirmOpen(true);
      },
    },
  ];

  return (
    <>
      <TableActionPopover items={menuItems}>
        <Button
          ref={triggerRef}
          variant="outline"
          className="size-5 [&_svg]:size-3 rounded-sm border-transparent group-hover:border-grayA-6"
          onClick={(e) => e.stopPropagation()}
        >
          <Dots className="group-hover:text-gray-12 text-gray-11" iconSize="sm-regular" />
        </Button>
      </TableActionPopover>

      <ConfirmPopover
        isOpen={isDeleteConfirmOpen}
        onOpenChange={setIsDeleteConfirmOpen}
        onConfirm={() => collection.envVars.delete(items.map((i) => i.id))}
        triggerRef={triggerRef}
        title="Confirm deletion"
        description={`This will permanently delete "${groupKey}" from ${items.length} environments (${environmentNames}). This action cannot be undone.`}
        confirmButtonText="Delete from all environments"
        cancelButtonText="Cancel"
        variant="danger"
      />

      <EnvVarGroupRenameDialog
        isOpen={isRenameOpen}
        onOpenChange={setIsRenameOpen}
        groupKey={groupKey}
        items={items}
      />
    </>
  );
}
