"use client";

import { cn } from "@/lib/utils";
import { Check, TriangleWarning, XMark } from "@unkey/icons";
import { Button, ConfirmPopover } from "@unkey/ui";
import { useRef, useState } from "react";

type SentinelPoliciesUnsavedBarProps = {
  hasPending: boolean;
  onSave: () => void;
  onDiscard: () => void;
};

export function SentinelPoliciesUnsavedBar({
  hasPending,
  onSave,
  onDiscard,
}: SentinelPoliciesUnsavedBarProps) {
  const [isDiscardConfirmOpen, setIsDiscardConfirmOpen] = useState(false);
  const discardButtonRef = useRef<HTMLButtonElement>(null);

  if (!hasPending) {
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
            <div className="items-center flex gap-2 text-gray-11 text-[13px] leading-6">
              <div className="flex items-center justify-center size-[22px] rounded-md bg-warning-3 border border-warning-6">
                <TriangleWarning iconSize="md-medium" className="text-warning-11" />
              </div>
              You have unsaved changes. They won't take effect until you save.
            </div>
            <div className="flex items-center gap-2">
              <Button
                variant="outline"
                size="sm"
                className="font-medium text-[13px] [&_svg]:size-3.5"
                onClick={() => setIsDiscardConfirmOpen(true)}
                ref={discardButtonRef}
              >
                <XMark iconSize="sm-medium" />
                Discard
              </Button>
              <Button
                variant="primary"
                size="sm"
                className="font-medium text-[13px] [&_svg]:size-3.5"
                onClick={onSave}
              >
                <Check iconSize="sm-medium" />
                Save
              </Button>
            </div>
          </div>
        </div>
      </div>

      <ConfirmPopover
        isOpen={isDiscardConfirmOpen}
        onOpenChange={setIsDiscardConfirmOpen}
        onConfirm={onDiscard}
        triggerRef={discardButtonRef}
        title="Discard unsaved changes"
        description="This will revert your unsaved edits to the last saved state. This action cannot be undone."
        confirmButtonText="Discard changes"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
}
