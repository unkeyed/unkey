"use client";

import { Button, ConfirmPopover, toast } from "@unkey/ui";
import {
  IconCloneOutline18,
  IconDotsOutline18,
  IconPenWriting3Outline18,
  IconTrashOutline18,
} from "nucleo-ui-outline-18";
import { useRef, useState } from "react";
import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { collection } from "@/lib/collections";

type EnvVarActionMenuProps = {
  envVarId: string;
  value: string;
  variableKey: string;
  type: "writeonly" | "recoverable";
  onEdit: () => void;
};

export function EnvVarActionMenu({
  envVarId,
  value,
  variableKey,
  type,
  onEdit,
}: EnvVarActionMenuProps) {
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const menuItems: MenuItem[] = [
    {
      id: "edit",
      label: "Edit",
      icon: <IconPenWriting3Outline18 className="size-4" />,
      onClick: (e) => {
        e.stopPropagation();
        onEdit();
      },
    },
    {
      id: "delete",
      label: "Delete",
      icon: <IconTrashOutline18 className="size-4" />,
      divider: true,
      onClick: (e) => {
        e.stopPropagation();
        setIsDeleteConfirmOpen(true);
      },
    },
    {
      id: "copy",
      label: "Copy to Clipboard",
      icon: <IconCloneOutline18 className="size-4" />,
      disabled: type === "writeonly",
      tooltip: type === "writeonly" ? "Write-only variables cannot be copied" : undefined,
      onClick: async (e) => {
        e.stopPropagation();
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
          ref={deleteButtonRef}
          variant="outline"
          className="size-5 [&_svg]:size-3 rounded-sm border-transparent group-hover:border-grayA-6"
          onClick={(e) => e.stopPropagation()}
        >
          <IconDotsOutline18 className="group-hover:text-gray-12 text-gray-11" />
        </Button>
      </TableActionPopover>

      {/* Stop clicks inside the popover (e.g. Cancel) from bubbling up to the
          row, which would otherwise open the edit form. */}
      <div onClick={(e) => e.stopPropagation()} onKeyDown={(e) => e.stopPropagation()}>
        <ConfirmPopover
          isOpen={isDeleteConfirmOpen}
          onOpenChange={setIsDeleteConfirmOpen}
          onConfirm={() => collection.envVars.delete([envVarId])}
          triggerRef={deleteButtonRef}
          title="Confirm deletion"
          description={`This will permanently delete "${variableKey}". This action cannot be undone.`}
          confirmButtonText="Delete variable"
          cancelButtonText="Cancel"
          variant="danger"
        />
      </div>
    </>
  );
}
