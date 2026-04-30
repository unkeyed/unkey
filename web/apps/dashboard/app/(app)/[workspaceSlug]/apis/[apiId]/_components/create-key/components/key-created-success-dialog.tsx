"use client";

import { Check, Key2 } from "@unkey/icons";
import { ConfirmPopover, Dialog, DialogContent, DialogTitle, VisuallyHidden } from "@unkey/ui";
import { type FC, useEffect, useRef, useState } from "react";
import { KeySecretSection } from "./key-secret-section";

interface KeyCreatedSuccessDialogProps {
  isOpen: boolean;
  onClose: () => void;
  keyData: { key: string } | null;
}

export const KeyCreatedSuccessDialog: FC<KeyCreatedSuccessDialogProps> = ({
  isOpen,
  onClose,
  keyData,
}) => {
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const popoverAnchorRef = useRef<HTMLDivElement>(null);

  // Prevent accidental tab/window close while the secret is visible
  useEffect(() => {
    if (!isOpen) {
      return;
    }

    const handleBeforeUnload = (e: BeforeUnloadEvent) => {
      e.preventDefault();
    };

    window.addEventListener("beforeunload", handleBeforeUnload);

    return () => {
      window.removeEventListener("beforeunload", handleBeforeUnload);
    };
  }, [isOpen]);

  if (!keyData) {
    return null;
  }

  const handleCloseAttempt = () => {
    setIsConfirmOpen(true);
  };

  const handleConfirmClose = () => {
    setIsConfirmOpen(false);
    onClose();
  };

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(open) => {
        if (!open) {
          handleCloseAttempt();
        }
      }}
    >
      <DialogContent
        className="drop-shadow-2xl transform-gpu border-grayA-4 overflow-hidden rounded-2xl! p-0 gap-0 w-full max-w-[760px] max-h-[90vh] overflow-y-auto"
        showCloseWarning
        onAttemptClose={handleCloseAttempt}
      >
        <VisuallyHidden asChild>
          <DialogTitle>Key Created</DialogTitle>
        </VisuallyHidden>
        <div className="bg-grayA-2 py-10 flex flex-col items-center justify-center w-full px-[120px]">
          <div className="py-4 mt-[30px]">
            <div className="flex gap-4">
              <div className="border border-grayA-4 rounded-[14px] size-14 opacity-35" />
              <div className="border border-grayA-4 rounded-[14px] size-14" />
              <div className="border border-grayA-4 rounded-[14px] size-14 flex items-center justify-center relative">
                <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute left-0 top-0" />
                <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute right-0 top-0" />
                <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute right-0 bottom-0" />
                <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute left-0 bottom-0" />
                <Key2 iconSize="2xl-thin" aria-hidden="true" focusable={false} />
                <div className="flex items-center justify-center border border-grayA-3 rounded-full bg-success-9 text-white size-[22px] absolute right-[-10px] top-[-10px]">
                  <Check iconSize="sm-bold" aria-hidden="true" focusable={false} />
                </div>
              </div>
              <div className="border border-grayA-4 rounded-[14px] size-14" />
              <div className="border border-grayA-4 rounded-[14px] size-14 opacity-35" />
            </div>
          </div>
          <div className="mt-5 flex flex-col gap-2 items-center">
            <div className="font-semibold text-gray-12 text-[16px] leading-[24px]">Key Created</div>
            <div
              className="text-gray-10 text-[13px] leading-[24px] text-center"
              ref={popoverAnchorRef}
            >
              You've successfully generated a new API key.
              <br /> Use this key to authenticate requests from your application.
            </div>
          </div>
          <div className="p-1 w-full my-8">
            <div className="h-px bg-grayA-3 w-full" />
          </div>
          <KeySecretSection keyValue={keyData.key} className="w-full" />
        </div>
        <ConfirmPopover
          isOpen={isConfirmOpen}
          onOpenChange={setIsConfirmOpen}
          onConfirm={handleConfirmClose}
          triggerRef={popoverAnchorRef}
          title="You won't see this secret key again!"
          description="Make sure to copy your secret key before closing. It cannot be retrieved later."
          confirmButtonText="Close anyway"
          cancelButtonText="Dismiss"
          variant="warning"
          popoverProps={{
            side: "right",
            align: "end",
            sideOffset: 5,
            alignOffset: 30,
            onOpenAutoFocus: (e) => e.preventDefault(),
          }}
        />
      </DialogContent>
    </Dialog>
  );
};
