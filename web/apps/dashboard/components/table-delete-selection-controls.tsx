"use client";
import { AnimatedCounter } from "@/components/api-keys-table/components/selection-controls";
import { MotionProvider } from "@/components/motion-provider";
import { IconTrashOutline18, IconXmarkOutline18 } from "@unkey/icons";
import { Button, ConfirmPopover, ControlsSelectionPortal } from "@unkey/ui";
import { AnimatePresence, m } from "framer-motion";
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
    <MotionProvider>
      <ControlsSelectionPortal>
        <AnimatePresence>
          {selectedCount > 0 && (
            <m.div
              key="selection-controls"
              className="flex min-w-0 flex-1 items-center justify-between gap-2"
              initial={{ opacity: 0 }}
              animate={{ opacity: 1, transition: { duration: 0.2, ease: "easeOut" } }}
              exit={{ opacity: 0, transition: { duration: 0.15, ease: "easeIn" } }}
            >
              <div className="flex items-center gap-2">
                <AnimatedCounter value={selectedCount} />
                <div className="text-gray-9 text-sm">selected</div>
              </div>
              <div className="flex shrink-0 items-center gap-2">
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
                  className="[&_svg]:size-[14px]"
                  onClick={onClearSelection}
                  aria-label="Clear selection"
                >
                  <IconXmarkOutline18 />
                </Button>
              </div>
            </m.div>
          )}
        </AnimatePresence>
      </ControlsSelectionPortal>

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
    </MotionProvider>
  );
};
