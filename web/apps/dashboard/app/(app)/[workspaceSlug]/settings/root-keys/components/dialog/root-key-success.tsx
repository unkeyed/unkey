"use client";

import { SecretKey } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/secret-key";
import { Check, CircleInfo, Key2 } from "@unkey/icons";
import { ConfirmPopover, Dialog, DialogContent, DialogTitle, VisuallyHidden } from "@unkey/ui";
import { ROOT_KEY_MESSAGES } from "./constants";
import { useRootKeySuccess } from "./hooks/use-root-key-success";

type RootKeySuccessProps = {
  keyValue?: string;
  onClose: () => void;
};

export const RootKeySuccess = ({ keyValue, onClose }: RootKeySuccessProps) => {
  const { isConfirmOpen, setIsConfirmOpen, dividerRef, handleCloseAttempt, handleConfirmClose } =
    useRootKeySuccess({
      onClose,
    });

  if (!keyValue) {
    return null;
  }

  return (
    <Dialog
      open={!!keyValue}
      onOpenChange={(open) => {
        if (!open) {
          handleCloseAttempt();
        }
      }}
    >
      <DialogContent
        className="drop-shadow-2xl transform-gpu border-grayA-4 overflow-hidden !rounded-2xl p-0 gap-0 w-full max-w-[760px] max-h-[90vh] overflow-y-auto"
        showCloseWarning
        onAttemptClose={handleCloseAttempt}
      >
        <VisuallyHidden asChild>
          <DialogTitle>{ROOT_KEY_MESSAGES.SUCCESS.ROOT_KEY_CREATED}</DialogTitle>
        </VisuallyHidden>
        <div className="bg-grayA-2 py-10 flex flex-col items-center justify-center w-full px-[120px]">
          <div className="py-4 mt-[30px]">
            <div className="flex gap-4">
              <div className="border border-grayA-4 rounded-[10px] size-14 opacity-35" />
              <div className="border border-grayA-4 rounded-[10px] size-14" />
              <div className="border border-grayA-4 rounded-[10px] size-14 flex items-center justify-center relative">
                <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute left-0 top-0" />
                <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute right-0 top-0" />
                <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute right-0 bottom-0" />
                <div className="border border-grayA-4 rounded-full border-dashed size-[24px] absolute left-0 bottom-0" />
                <Key2 iconSize="2xl-thin" aria-hidden="true" focusable={false} />
                <div className="flex items-center justify-center border border-grayA-3 rounded-full bg-success-9 text-white size-[22px] absolute right-[-10px] top-[-10px]">
                  <Check iconSize="sm-bold" aria-hidden="true" focusable={false} />
                </div>
              </div>
              <div className="border border-grayA-4 rounded-[10px] size-14" />
              <div className="border border-grayA-4 rounded-[10px] size-14 opacity-35" />
            </div>
          </div>
          <div className="mt-5 flex flex-col gap-2 items-center">
            <div className="font-semibold text-gray-12 text-[16px] leading-[24px]">
              {ROOT_KEY_MESSAGES.SUCCESS.ROOT_KEY_CREATED}
            </div>
            <div className="text-gray-10 text-[13px] leading-[24px] text-center" ref={dividerRef}>
              {ROOT_KEY_MESSAGES.SUCCESS.ROOT_KEY_GENERATED}
            </div>
          </div>
          <div className="p-1 w-full my-8">
            <div className="h-[1px] bg-grayA-3 w-full" />
          </div>

          <div className="flex flex-col gap-2 items-start w-full">
            <div className="text-gray-12 text-sm font-semibold">
              {ROOT_KEY_MESSAGES.UI.ROOT_KEY}
            </div>
            <SecretKey
              value={keyValue}
              title={ROOT_KEY_MESSAGES.UI.ROOT_KEY}
              className="bg-white dark:bg-black "
            />
            <div className="text-gray-9 text-[13px] flex items-center gap-1.5">
              <CircleInfo
                className="text-accent-9"
                iconSize="sm-regular"
                aria-hidden="true"
                focusable={false}
              />
              <span className="flex items-center gap-1">
                {ROOT_KEY_MESSAGES.UI.COPY_SAVE_KEY}
                <a
                  href="https://www.unkey.com/docs/security/recovering-keys"
                  target="_blank"
                  rel="noopener noreferrer"
                  className="text-info-11 hover:underline"
                >
                  {ROOT_KEY_MESSAGES.UI.LEARN_MORE}
                </a>
              </span>
            </div>
          </div>
          <div className="mt-6">
            <div className="mt-4 text-center text-gray-10 text-xs leading-6">
              {ROOT_KEY_MESSAGES.UI.ALL_SET}
            </div>
          </div>
        </div>
        <ConfirmPopover
          isOpen={isConfirmOpen}
          onOpenChange={setIsConfirmOpen}
          onConfirm={handleConfirmClose}
          triggerRef={dividerRef}
          title={ROOT_KEY_MESSAGES.WARNING.WONT_SEE_AGAIN}
          description={ROOT_KEY_MESSAGES.WARNING.COPY_BEFORE_CLOSING}
          confirmButtonText={ROOT_KEY_MESSAGES.UI.CLOSE_ANYWAY}
          cancelButtonText={ROOT_KEY_MESSAGES.UI.DISMISS}
          variant="warning"
          popoverProps={{
            side: "right",
            align: "end",
            sideOffset: 5,
            alignOffset: 30,
            // Let the popover auto-focus internally to meet WCAG 2.4.3/2.1.1
            // onOpenAutoFocus: (e) => e.preventDefault(),
          }}
        />
      </DialogContent>
    </Dialog>
  );
};
