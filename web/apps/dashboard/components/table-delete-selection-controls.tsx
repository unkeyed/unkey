"use client";
import { IconTrashOutline18 } from "@unkey/icons";
import { ConfirmPopover, SelectionBar, SelectionBarAction } from "@unkey/ui";
import { useRef, useState } from "react";

type TableDeleteSelectionControlsProps = {
  selectedCount: number;
  onClearSelection: () => void;
  onConfirmDelete: () => void;
  isDeleting: boolean;
  // Singular noun in lower case ("role", "permission"). Used in labels and confirm copy.
  singular: string;
  // Plural noun in lower case ("roles", "permissions"). Used in the delete button label.
  plural: string;
};

export function TableDeleteSelectionControls({
  selectedCount,
  onClearSelection,
  onConfirmDelete,
  isDeleting,
  singular,
  plural,
}: TableDeleteSelectionControlsProps) {
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  return (
    <>
      <SelectionBar count={selectedCount} onClear={onClearSelection}>
        <SelectionBarAction
          disabled={isDeleting}
          loading={isDeleting}
          onClick={() => setIsDeleteConfirmOpen(true)}
          ref={deleteButtonRef}
          icon={<IconTrashOutline18 />}
          label={`Delete ${plural}`}
        />
      </SelectionBar>

      <ConfirmPopover
        isOpen={isDeleteConfirmOpen}
        onOpenChange={setIsDeleteConfirmOpen}
        onConfirm={onConfirmDelete}
        triggerRef={deleteButtonRef}
        title={`Confirm ${singular} deletion`}
        description={`This action is irreversible. All data associated with ${
          selectedCount > 1 ? `these ${plural}` : `this ${singular}`
        } will be permanently deleted.`}
        confirmButtonText={`Delete ${singular}${selectedCount > 1 ? "s" : ""}`}
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
}
