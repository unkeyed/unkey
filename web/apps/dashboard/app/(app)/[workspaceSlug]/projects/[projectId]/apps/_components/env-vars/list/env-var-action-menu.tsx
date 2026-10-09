"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { collection } from "@/lib/collections";
import type { EnvVar } from "@/lib/collections/deploy/env-vars";
import {
  IconCloneOutline18,
  IconDotsOutline18,
  IconPenWriting3Outline18,
  IconTrashOutline18,
} from "@unkey/icons";
import { Button, ConfirmPopover, toast } from "@unkey/ui";
import { useRef, useState } from "react";

export function EnvVarActionMenu({ envVar, onEdit }: { envVar: EnvVar; onEdit: () => void }) {
  const { id: envVarId, value, key: variableKey, type } = envVar;
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const triggerRef = useRef<HTMLButtonElement>(null);

  const menuItems: MenuItem[] = [
    {
      id: "edit",
      label: "Edit",
      icon: <IconPenWriting3Outline18 className="size-3.5" />,
      onClick: onEdit,
    },
    {
      id: "delete",
      label: "Delete",
      icon: <IconTrashOutline18 className="size-3.5" />,
      divider: true,
      onClick: () => setIsDeleteConfirmOpen(true),
    },
    {
      id: "copy",
      label: "Copy to Clipboard",
      icon: <IconCloneOutline18 className="size-3.5" />,
      disabled: type === "writeonly",
      tooltip: type === "writeonly" ? "Write-only variables cannot be copied" : undefined,
      onClick: async () => {
        try {
          await navigator.clipboard.writeText(`${variableKey}=${value}`);
          toast.success("Copied to clipboard");
        } catch {
          toast.error("Failed to copy to clipboard");
        }
      },
    },
  ];

  return (
    <>
      <TableActionPopover items={menuItems}>
        <Button
          ref={triggerRef}
          variant="outline"
          className="size-5 [&_svg]:size-3 rounded-sm border-transparent group-hover:border-strong"
        >
          <IconDotsOutline18 className="group-hover:text-gray-12 text-gray-11" />
        </Button>
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
