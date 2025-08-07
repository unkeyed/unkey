"use client";

import { SecretKey } from "@/app/(app)/apis/[apiId]/_components/create-key/components/secret-key";
import { ConfirmPopover } from "@/components/confirmation-popover";
import { ArrowRight, Check, CircleInfo, Key2 } from "@unkey/icons";
import {
  Button,
  Code,
  CopyButton,
  Dialog,
  DialogContent,
  InfoTooltip,
  VisibleButton,
} from "@unkey/ui";
import { ROOT_KEY_CONSTANTS } from "./constants";
import { useRootKeySuccess } from "./hooks/use-root-key-success";

type RootKeySuccessProps = {
  keyValue?: string;
  keyId?: string;
  name?: string;
  onClose: () => void;
};

export const RootKeySuccess = ({ keyValue, keyId, name, onClose }: RootKeySuccessProps) => {
  const {
    showKeyInSnippet,
    setShowKeyInSnippet,
    isConfirmOpen,
    setIsConfirmOpen,
    dividerRef,
    handleCloseAttempt,
    handleConfirmClose,
    handleDialogOpenChange,
    snippet,
    maskedKey,
  } = useRootKeySuccess({
    keyValue,
    onClose,
  });

  if (!keyValue || !keyId) {
    return null;
  }

  return (
    <Dialog open={!!keyValue} onOpenChange={handleDialogOpenChange}>
      <DialogContent
        className="drop-shadow-2xl border-gray-4 overflow-hidden !rounded-2xl p-0 gap-0 min-w-[760px] max-h-[90vh] overflow-y-auto"
        showCloseWarning
        onAttemptClose={() => handleCloseAttempt("close")}
      >
        <>
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
                  <Key2 size="2xl-thin" />
                  <div className="flex items-center justify-center border border-grayA-3 rounded-full bg-success-9 text-white size-[22px] absolute right-[-10px] top-[-10px]">
                    <Check size="sm-bold" />
                  </div>
                </div>
                <div className="border border-grayA-4 rounded-[10px] size-14" />
                <div className="border border-grayA-4 rounded-[10px] size-14 opacity-35" />
              </div>
            </div>
            <div className="mt-5 flex flex-col gap-2 items-center">
              <div className="font-semibold text-gray-12 text-[16px] leading-[24px]">
                Root Key Created
              </div>
              <div className="text-gray-10 text-[13px] leading-[24px] text-center" ref={dividerRef}>
                You've successfully generated a new Root key.
              </div>
            </div>
            <div className="p-1 w-full my-8">
              <div className="h-[1px] bg-grayA-3 w-full" />
            </div>
            <div className="flex flex-col gap-2 items-start w-full">
              <div className="text-gray-12 text-sm font-semibold">Key Details</div>
              <div className="bg-white dark:bg-black border rounded-xl border-grayA-5 px-6 w-full">
                <div className="flex gap-6 items-center">
                  <div className="bg-grayA-5 text-gray-12 size-5 flex items-center justify-center rounded ">
                    <Key2 size="sm-regular" />
                  </div>
                  <div className="flex flex-col gap-1 py-6">
                    <div className="text-accent-12 text-xs font-mono">{keyId}</div>
                    <InfoTooltip
                      content={name ?? ROOT_KEY_CONSTANTS.UNNAMED_KEY}
                      position={{ side: "bottom", align: "center" }}
                      asChild
                      disabled={!name}
                      variant="inverted"
                    >
                      <div className="text-accent-9 text-xs max-w-[160px] truncate">
                        {name ?? ROOT_KEY_CONSTANTS.UNNAMED_KEY}
                      </div>
                    </InfoTooltip>
                  </div>
                  <Button
                    variant="outline"
                    className="ml-auto font-medium text-[13px] text-gray-12"
                    onClick={() => handleCloseAttempt("go-to-details")}
                  >
                    See key details <ArrowRight size="sm-regular" />
                  </Button>
                </div>
              </div>
            </div>
            <div className="flex flex-col gap-2 items-start w-full mt-6">
              <div className="text-gray-12 text-sm font-semibold">Key Secret</div>
              <SecretKey value={keyValue} title="API Key" className="bg-white dark:bg-black " />
              <div className="text-gray-9 text-[13px] flex items-center gap-1.5">
                <CircleInfo className="text-accent-9" size="sm-regular" />
                <span>
                  Copy and save this key secret as it won't be shown again.{" "}
                  <a
                    href="https://www.unkey.com/docs/security/recovering-keys"
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-info-11 hover:underline"
                  >
                    Learn more
                  </a>
                </span>
              </div>
            </div>
            <div className="flex flex-col gap-2 items-start w-full mt-8">
              <div className="text-gray-12 text-sm font-semibold">Try It Out</div>
              <Code
                visibleButton={
                  <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
                }
                copyButton={<CopyButton value={snippet} />}
              >
                {showKeyInSnippet ? snippet : snippet.replace(keyValue, maskedKey)}
              </Code>
            </div>
            <div className="mt-6">
              <div className="mt-4 text-center text-gray-10 text-xs leading-6">
                All set! You can now create another key or explore the docs to learn more
              </div>
            </div>
          </div>
          <ConfirmPopover
            isOpen={isConfirmOpen}
            onOpenChange={setIsConfirmOpen}
            onConfirm={handleConfirmClose}
            triggerRef={dividerRef}
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
        </>
      </DialogContent>
    </Dialog>
  );
};
