"use client";
import { AnimatedCounter } from "@/components/api-keys-table/components/selection-controls";
import { Trash, XMark } from "@unkey/icons";
import { Button, ConfirmPopover } from "@unkey/ui";
import { AnimatePresence, motion } from "framer-motion";
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
      <AnimatePresence>
        {selectedCount > 0 && (
          <motion.div
            key="selection-controls"
            className="border-b border-grayA-3 w-full overflow-hidden"
            initial={{ opacity: 0, height: 0 }}
            animate={{
              opacity: 1,
              height: "auto",
              transition: {
                height: { duration: 0.3, ease: "easeOut" },
                opacity: { duration: 0.3, ease: "easeOut" },
              },
            }}
            exit={{
              opacity: 0,
              height: 0,
              transition: {
                opacity: { duration: 0.2, ease: "easeIn" },
                height: { duration: 0.3, ease: "easeIn" },
              },
            }}
          >
            <div className="flex justify-between items-center w-full p-[18px]">
              <div className="items-center flex gap-2">
                <AnimatedCounter value={selectedCount} />
                <div className="text-accent-9 text-[13px] leading-6">selected</div>
              </div>
              <div className="flex items-center gap-2">
                <Button
                  variant="outline"
                  size="sm"
                  className="text-gray-12 font-medium text-[13px]"
                  disabled={isDeleting}
                  loading={isDeleting}
                  onClick={() => setIsDeleteConfirmOpen(true)}
                  ref={deleteButtonRef}
                >
                  <Trash iconSize="sm-regular" />
                  Delete {plural}
                </Button>
                <Button
                  size="icon"
                  variant="ghost"
                  className="[&_svg]:size-[14px] ml-3"
                  onClick={onClearSelection}
                >
                  <XMark />
                </Button>
              </div>
            </div>
          </motion.div>
        )}
      </AnimatePresence>

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
