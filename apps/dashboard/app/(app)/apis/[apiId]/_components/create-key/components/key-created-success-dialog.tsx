"use client";

import { ConfirmPopover } from "@/components/confirmation-popover";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Dialog, DialogContent } from "@/components/ui/dialog";
import { toast } from "@/components/ui/toaster";
import * as PopoverPrimitive from "@radix-ui/react-popover";
import {
  ArrowRight,
  Check,
  CircleInfo,
  Eye,
  EyeSlash,
  Key2,
  Plus,
  TriangleWarning2,
} from "@unkey/icons";
import { InfoTooltip } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { useRef, useState } from "react";
import { UNNAMED_KEY } from "../create-key.constants";
import { SecretKey } from "./secret-key";

export const KeyCreatedSuccessDialog = ({
  isOpen,
  onClose,
  keyData,
  apiId,
  keyspaceId,
  onCreateAnother,
}: {
  isOpen: boolean;
  onClose: () => void;
  keyData: { key: string; id: string; name?: string } | null;
  apiId: string;
  keyspaceId?: string | null;
  onCreateAnother?: () => void;
}) => {
  const [showKeyInSnippet, setShowKeyInSnippet] = useState(false);
  const [isConfirmOpen, setIsConfirmOpen] = useState(false);
  const [isCreateAnother, setIsCreateAnother] = useState(false);
  const xButtonRef = useRef<HTMLButtonElement>(null);
  const dividerRef = useRef<HTMLDivElement>(null);

  if (!keyData) {
    return null;
  }

  const split = keyData.key.split("_") ?? [];
  const maskedKey =
    split.length >= 2
      ? `${split.at(0)}_${"*".repeat(split.at(1)?.length ?? 0)}`
      : "*".repeat(split.at(0)?.length ?? 0);

  const snippet = `curl -XPOST '${
    process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "https://api.unkey.dev"
  }/v1/keys.verifyKey' \\
  -H 'Content-Type: application/json' \\
  -d '{
    "key": "${keyData.key}",
    "apiId": "${apiId}"
  }'`;

  const handleAttemptClose = (shouldCreateAnother = false) => {
    setIsCreateAnother(shouldCreateAnother);
    setIsConfirmOpen(true);
  };

  const handleConfirmAndClose = () => {
    setIsConfirmOpen(false);
    onClose();
    if (isCreateAnother) {
      onCreateAnother?.();
    }
  };

  return (
    <Dialog
      open={isOpen}
      onOpenChange={(open) => {
        if (!open) {
          onClose();
        }
      }}
    >
      <DialogContent
        className="drop-shadow-2xl border-gray-4 overflow-hidden !rounded-2xl p-0 gap-0 min-w-[760px] max-h-[90vh] overflow-y-auto"
        showCloseWarning
        onAttemptClose={handleAttemptClose}
        xButtonRef={xButtonRef}
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
                Key Created
              </div>
              <div className="text-gray-10 text-[13px] leading-[24px] text-center" ref={dividerRef}>
                You've successfully generated a new API key.
                <br /> Use this key to authenticate requests from your application.
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
                    <div className="text-accent-12 text-xs font-mono">{keyData.id}</div>
                    <InfoTooltip
                      content={keyData.name}
                      position={{ side: "bottom", align: "center" }}
                      asChild
                      disabled={!keyData.name}
                    >
                      <div className="text-accent-9 text-xs max-w-[160px] truncate">
                        {keyData.name ?? UNNAMED_KEY}
                      </div>
                    </InfoTooltip>
                  </div>
                  <Button
                    variant="outline"
                    className="ml-auto font-medium text-[13px] text-gray-12"
                    onClick={() => {
                      if (!keyspaceId) {
                        toast.error("Failed to Create Key", {
                          description: "An unexpected error occurred. Please try again later.",
                          action: {
                            label: "Contact Support",
                            onClick: () => window.open("https://support.unkey.dev", "_blank"),
                          },
                        });
                        return;
                      }
                      window.location.href = `/apis/${apiId}/keys/${keyspaceId}/${keyData.id}`;
                    }}
                  >
                    See key details <ArrowRight size="sm-regular" />
                  </Button>
                </div>
              </div>
            </div>
            <div className="flex flex-col gap-2 items-start w-full mt-6">
              <div className="text-gray-12 text-sm font-semibold">Key Secret</div>
              <SecretKey value={keyData.key} title="API Key" className="bg-white dark:bg-black " />
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
              <div className="w-full px-4 py-2 bg-white dark:bg-black border rounded-xl border-grayA-5">
                <div className="flex items-start justify-between w-full gap-4 bg-transparent">
                  <div className="mt-2 overflow-x-auto w-full min-w-0">
                    <pre className="ph-no-capture whitespace-pre-wrap break-all text-[11px] pr-2">
                      {showKeyInSnippet ? snippet : snippet.replace(keyData.key, maskedKey)}
                    </pre>
                  </div>
                  <div className="flex items-center justify-between gap-2 mt-1 flex-shrink-0">
                    <Button
                      variant="outline"
                      size="icon"
                      className="bg-grayA-3 transition-all"
                      onClick={() => setShowKeyInSnippet(!showKeyInSnippet)}
                    >
                      {showKeyInSnippet ? <EyeSlash /> : <Eye />}
                    </Button>
                    <Button variant="outline" size="icon" className="bg-grayA-3">
                      <div className="flex items-center justify-center">
                        <CopyButton value={snippet} />
                      </div>
                    </Button>
                  </div>
                </div>
              </div>
            </div>
            <div className="mt-6">
              <div className="mt-4 text-center text-gray-10 text-xs leading-6">
                All set! You can now create another key or explore the docs to learn more
              </div>
              <div className="flex gap-3 mt-4 items-center justify-center w-full">
                <Button
                  variant="outline"
                  className="font-medium text-[13px] text-gray-12"
                  onClick={() => {
                    handleAttemptClose(true);
                  }}
                >
                  <Plus size="sm-regular" />
                  Create another key
                </Button>
                {/* INFO: We'll add this back soon */}
                {/* <Button */}
                {/*   variant="outline" */}
                {/*   className="font-medium text-[13px] text-gray-12" */}
                {/*   disabled */}
                {/*   title="Coming soon!" */}
                {/* > */}
                {/*   <Bookmark size="sm-regular" /> */}
                {/*   Learn best practices */}
                {/* </Button> */}
              </div>
            </div>
          </div>
          <ConfirmPopover
            isOpen={isConfirmOpen}
            onOpenChange={setIsConfirmOpen}
            onConfirm={handleConfirmAndClose}
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
