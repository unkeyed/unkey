"use client";
import { AnimatedCounter } from "@/components/api-keys-table/components/selection-controls";
import { Presence } from "@/components/presence";
import { IconTrashOutline18, IconXmarkOutline18 } from "@unkey/icons";
import { Button, ConfirmPopover } from "@unkey/ui";
import { cn } from "cn";
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

// Shared selection bar for tables that support bulk-delete only. Renders the
// selection count, a Delete button with confirm popover, and a clear-selection
// X. Callers own the delete mutation and selection state.
export const TableDeleteSelectionControls = ({
  selectedCount,
  onClearSelection,
  onConfirmDelete,
  isDeleting,
  singular,
  plural,
}: TableDeleteSelectionControlsProps) => {
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  return (
    <>
      <Presence present={selectedCount > 0} exitMs={300}>
        {({ closing }) => (
          <div
            className={cn(
              "border-b w-full grid",
              closing
                ? "grid-rows-[0fr] opacity-0 [transition:grid-template-rows_300ms_ease-in,opacity_200ms_ease-in]"
                : "grid-rows-[1fr] opacity-100 [transition:grid-template-rows_300ms_ease-out,opacity_300ms_ease-out] starting:grid-rows-[0fr] starting:opacity-0",
            )}
          >
            <div className="min-h-0 overflow-hidden">
            <div className="flex justify-between items-center w-full p-[18px]">
              <div className="items-center flex gap-2">
                <AnimatedCounter value={selectedCount} />
                <div className="text-gray-9 text-sm leading-6">selected</div>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  className="text-gray-12 font-medium text-sm"
                  disabled={isDeleting}
                  loading={isDeleting}
                  onClick={() => setIsDeleteConfirmOpen(true)}
                  ref={deleteButtonRef}
                >
                  <IconTrashOutline18 />
                  Delete {plural}
                </Button>
                <Button
                  size="icon"
                  variant="ghost"
                  className="[&_svg]:size-[14px] ml-3"
                  onClick={onClearSelection}
                  aria-label="Clear selection"
                >
                  <IconXmarkOutline18 />
                </Button>
              </div>
            </div>
            </div>
          </div>
        )}
      </Presence>

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
};
