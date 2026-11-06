"use client";

import {
  type MenuItem,
  TableActionPopover,
  TableActionPopoverDefaultTrigger,
} from "@/components/logs/table-action.popover";
import { collection } from "@/lib/collections";
import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import { IconCloneOutline18, IconPenWriting3Outline18, IconTrashOutline18 } from "@unkey/icons";
import { ConfirmPopover, toast } from "@unkey/ui";
import { useRef, useState } from "react";

export function EnvVarActionMenu({ envVar, onEdit }: { envVar: EnvVar; onEdit: () => void }) {
  const { id: envVarId, value, key: variableKey, type } = envVar;
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);

  const isCopyable = type !== "writeonly";

  const menuItems: MenuItem[] = [
    {
      id: "edit",
      label: "Edit",
      icon: <IconPenWriting3Outline18 className="size-3.5" />,
      divider: !isCopyable,
      onClick: onEdit,
    },
    ...(isCopyable
      ? [
          {
            id: "copy",
            label: "Copy to Clipboard",
            icon: <IconCloneOutline18 className="size-3.5" />,
            className: "mt-1",
            divider: true,
            onClick: async () => {
              try {
                await navigator.clipboard.writeText(`${variableKey}=${value}`);
                toast.success("Copied to clipboard");
              } catch {
                toast.error("Failed to copy to clipboard");
              }
            },
          },
        ]
      : []),
    {
      id: "delete",
      label: "Delete",
      icon: <IconTrashOutline18 className="size-3.5 text-error-11" />,
      className: "text-error-11 hover:bg-error-3 focus:bg-error-3",
      onClick: () => setIsDeleteConfirmOpen(true),
    },
  ];

  return (
    <>
      <TableActionPopover items={menuItems} contentClassName="w-48 min-w-0">
        <TableActionPopoverDefaultTrigger ref={triggerRef} />
      </TableActionPopover>

      <ConfirmPopover
        isOpen={isDeleteConfirmOpen}
        onOpenChange={setIsDeleteConfirmOpen}
        onConfirm={() => collection.envVars.delete([envVarId])}
        triggerRef={triggerRef}
        title="Confirm deletion"
        description={`This will permanently delete "${variableKey}". This action cannot be undone.`}
        confirmButtonText="Delete variable"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
}
