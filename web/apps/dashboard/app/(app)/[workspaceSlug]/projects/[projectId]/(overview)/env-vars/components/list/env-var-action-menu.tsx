"use client";

import { type MenuItem, TableActionPopover } from "@/components/logs/table-action.popover";
import { collection } from "@/lib/collections";
import { trpc } from "@/lib/trpc/client";
import { Clone, Dots, PenWriting3, Trash } from "@unkey/icons";
import { Button, ConfirmPopover, toast } from "@unkey/ui";
import { useRef, useState } from "react";

type EnvVarActionMenuProps = {
  envVarId: string;
  variableKey: string;
  type: "writeonly" | "recoverable";
  onEdit: () => void;
};

export function EnvVarActionMenu({ envVarId, variableKey, type, onEdit }: EnvVarActionMenuProps) {
  const decryptMutation = trpc.deploy.envVar.decrypt.useMutation();
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const menuItems: MenuItem[] = [
    {
      id: "edit",
      label: "Edit",
      icon: <PenWriting3 iconSize="md-regular" />,
      onClick: (e) => {
        e.stopPropagation();
        onEdit();
      },
    },
    {
      id: "delete",
      label: "Delete",
      icon: <Trash iconSize="md-regular" />,
      divider: true,
      onClick: (e) => {
        e.stopPropagation();
        setIsDeleteConfirmOpen(true);
      },
    },
    {
      id: "copy",
      label: "Copy to Clipboard",
      icon: <Clone iconSize="md-regular" />,
      disabled: type === "writeonly",
      tooltip: type === "writeonly" ? "Write-only variables cannot be copied" : undefined,
      onClick: async (e) => {
        e.stopPropagation();
        try {
          const result = await decryptMutation.mutateAsync({ envVarId });
          navigator.clipboard.writeText(`${variableKey}=${result.value}`);
          toast.success("Copied to clipboard");
        } catch {
          toast.error("Failed to decrypt value");
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
          <Dots className="group-hover:text-gray-12 text-gray-11" iconSize="sm-regular" />
        </Button>
      </TableActionPopover>

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
    </>
  );
}
