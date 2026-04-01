"use client";

import { cn } from "@/lib/utils";
import { Trash, XMark } from "@unkey/icons";
import { Button, ConfirmPopover } from "@unkey/ui";
import { useRef, useState } from "react";

type EnvVarSelectionBarProps = {
  selectedCount: number;
  onDelete: () => void;
  onClearSelection: () => void;
  isDeleting: boolean;
};

export function EnvVarSelectionBar({
  selectedCount,
  onDelete,
  onClearSelection,
  isDeleting,
}: EnvVarSelectionBarProps) {
  const [isDeleteConfirmOpen, setIsDeleteConfirmOpen] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  if (selectedCount === 0) {
    return null;
  }

  return (
    <>
      <div className="sticky bottom-5 flex justify-center z-10 pointer-events-none">
        <div
          className={cn(
            "w-[740px] border bg-gray-1 dark:bg-black border-gray-6 min-h-[60px] flex items-center justify-center rounded-[10px] drop-shadow-lg shadow-sm pointer-events-auto",
            "animate-fade-slide-in",
          )}
        >
          <div className="flex justify-between items-center w-full p-[18px]">
            <div className="items-center flex gap-2">
              <AnimatedCounter value={selectedCount} />
              <div className="text-gray-11 text-[13px] leading-6">selected</div>
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                className="font-medium text-[13px] [&_svg]:size-3.5"
                disabled={isDeleting}
                loading={isDeleting}
                onClick={() => setIsDeleteConfirmOpen(true)}
                ref={deleteButtonRef}
              >
                <Trash iconSize="sm-medium" />
                Delete
              </Button>
              <Button
                size="icon"
                variant="ghost"
                className="[&_svg]:size-3.5 ml-3"
                onClick={onClearSelection}
              >
                <XMark iconSize="sm-medium" />
              </Button>
            </div>
          </div>
        </div>
      </div>

      <ConfirmPopover
        isOpen={isDeleteConfirmOpen}
        onOpenChange={setIsDeleteConfirmOpen}
        onConfirm={onDelete}
        triggerRef={deleteButtonRef}
        title="Confirm deletion"
        description={`This will permanently delete ${selectedCount} environment variable${selectedCount > 1 ? "s" : ""}. This action cannot be undone.`}
        confirmButtonText={`Delete variable${selectedCount > 1 ? "s" : ""}`}
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
}

function AnimatedCounter({ value }: { value: number }) {
  return (
    <div
      key={`counter-${value}`}
      className={cn(
        "size-[18px] text-[11px] leading-6 ring-2 ring-gray-6 flex items-center justify-center font-medium overflow-hidden p-2 text-white dark:text-black bg-accent-12 hover:bg-accent-12/90 focus:hover:bg-accent-12 rounded-md border border-grayA-4",
        "animate-bounce-in",
      )}
    >
      <span className="flex items-center justify-center">{value}</span>
    </div>
  );
}
