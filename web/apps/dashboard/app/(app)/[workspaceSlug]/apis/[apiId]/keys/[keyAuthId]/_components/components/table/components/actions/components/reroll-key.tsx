import { KeySecretSection } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/key-secret-section";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { Refresh3 } from "@unkey/icons";
import {
  Button,
  ConfirmPopover,
  Dialog,
  DialogContainer,
  DialogContent,
  DialogTitle,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useRerollKey } from "./hooks/use-reroll-key";
import { KeyInfo } from "./key-info";

const EXPIRATION_OPTIONS = [
  { label: "Immediately", value: "0" },
  { label: "1 hour", value: "3600000" },
  { label: "24 hours", value: "86400000" },
  { label: "7 days", value: "604800000" },
  { label: "30 days", value: "2592000000" },
] as const;

type RerollKeyProps = { keyDetails: KeyDetails } & ActionComponentProps;

export const RerollKey = ({ keyDetails, isOpen, onClose }: RerollKeyProps) => {
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const [newKeyData, setNewKeyData] = useState<{ keyId: string; key: string } | null>(null);
  const [isSuccessConfirmOpen, setIsSuccessConfirmOpen] = useState(false);
  const rerollButtonRef = useRef<HTMLButtonElement>(null);
  const successDividerRef = useRef<HTMLDivElement>(null);

  const { control, watch } = useForm({
    defaultValues: {
      expiration: "0",
    },
  });

  const expiration = watch("expiration");

  const rerollKey = useRerollKey((data) => {
    setNewKeyData(data);
  });

  const handleDialogOpenChange = (open: boolean) => {
    if (isConfirmPopoverOpen) {
      if (!open) {
        return;
      }
    } else {
      if (!open) {
        onClose();
      }
    }
  };

  const handleRerollButtonClick = () => {
    setIsConfirmPopoverOpen(true);
  };

  const performReroll = async () => {
    try {
      setIsLoading(true);
      await rerollKey.mutateAsync({
        keyId: keyDetails.id,
        expiration: Number.parseInt(expiration, 10),
      });
    } catch {
      // useRerollKey already shows a toast
    } finally {
      setIsLoading(false);
    }
  };

  const handleSuccessClose = () => {
    setIsSuccessConfirmOpen(false);
    setNewKeyData(null);
    rerollKey.invalidateKeys();
    onClose();
  };

  const handleSuccessDialogOpenChange = (open: boolean) => {
    if (!open) {
      setIsSuccessConfirmOpen(true);
    }
  };

  // Show success state with the new key
  if (newKeyData) {
    return (
      <Dialog open={true} onOpenChange={handleSuccessDialogOpenChange}>
        <DialogContent
          className="drop-shadow-2xl transform-gpu border-gray-4 rounded-2xl! p-0 gap-0 min-w-[560px] max-h-[90vh] overflow-auto [&::-webkit-scrollbar]:hidden [-ms-overflow-style:none] [scrollbar-width:none]"
          showCloseWarning
          onAttemptClose={() => setIsSuccessConfirmOpen(true)}
        >
          <>
            <DialogTitle className="sr-only">Key Rerolled Successfully</DialogTitle>
            <div className="bg-grayA-2 py-6 flex flex-col items-center justify-center w-full px-12 overflow-auto">
              <div className="py-4">
                <div className="flex items-center justify-center">
                  <div className="border border-grayA-4 rounded-[14px] size-14 flex items-center justify-center relative">
                    <Refresh3 iconSize="2xl-thin" />
                    <div className="flex items-center justify-center border border-grayA-3 rounded-full bg-success-9 text-white size-[22px] absolute right-[-10px] top-[-10px]">
                      <svg
                        width="12"
                        height="12"
                        viewBox="0 0 12 12"
                        fill="none"
                        xmlns="http://www.w3.org/2000/svg"
                      >
                        <path
                          d="M2.5 6L5 8.5L9.5 3.5"
                          stroke="currentColor"
                          strokeWidth="1.5"
                          strokeLinecap="round"
                          strokeLinejoin="round"
                        />
                      </svg>
                    </div>
                  </div>
                </div>
              </div>
              <div className="mt-2 flex flex-col gap-2 items-center">
                <div className="font-semibold text-gray-12 text-[16px] leading-[24px]">
                  Key Rerolled
                </div>
                <div
                  className="text-gray-10 text-[13px] leading-[24px] text-center"
                  ref={successDividerRef}
                >
                  Your key has been successfully rerolled.
                  <br /> The old key will{" "}
                  {expiration === "0"
                    ? "be revoked immediately"
                    : `expire in ${EXPIRATION_OPTIONS.find((o) => o.value === expiration)?.label?.toLowerCase()}`}
                  .
                </div>
              </div>
              <div className="p-1 w-full my-4">
                <div className="h-px bg-grayA-3 w-full" />
              </div>
              <KeySecretSection
                keyValue={newKeyData.key}
                apiId=""
                className="w-full"
                secretKeyClassName="bg-white dark:bg-black overflow-x-auto"
              />
              <div className="mt-6">
                <Button
                  variant="outline"
                  className="font-medium text-[13px] text-gray-12"
                  onClick={() => setIsSuccessConfirmOpen(true)}
                >
                  Done
                </Button>
              </div>
            </div>
            <ConfirmPopover
              isOpen={isSuccessConfirmOpen}
              onOpenChange={setIsSuccessConfirmOpen}
              onConfirm={handleSuccessClose}
              triggerRef={successDividerRef}
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
  }

  return (
    <>
      <DialogContainer
        isOpen={isOpen}
        subTitle="Generate a new key while preserving all configuration"
        onOpenChange={handleDialogOpenChange}
        title="Reroll key"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="button"
              variant="primary"
              color="danger"
              size="xlg"
              className="w-full rounded-lg"
              disabled={isLoading}
              loading={isLoading}
              onClick={handleRerollButtonClick}
              ref={rerollButtonRef}
            >
              Reroll key
            </Button>
            <div className="text-gray-9 text-xs">A new key will be generated immediately</div>
          </div>
        }
      >
        <KeyInfo keyDetails={keyDetails} />
        <div className="py-1 my-2">
          <div className="h-px bg-grayA-3 w-full" />
        </div>
        <div className="flex flex-col gap-1.5">
          <label htmlFor="expiration-select" className="text-gray-11 text-[13px] font-medium">
            Old key expiration
          </label>
          <Controller
            name="expiration"
            control={control}
            render={({ field }) => (
              <Select value={field.value} onValueChange={field.onChange}>
                <SelectTrigger id="expiration-select">
                  <SelectValue placeholder="Select expiration" />
                </SelectTrigger>
                <SelectContent>
                  {EXPIRATION_OPTIONS.map((option) => (
                    <SelectItem key={option.value} value={option.value}>
                      {option.label}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          />
          <p className="text-gray-9 text-xs">
            How long the original key remains valid after rerolling
          </p>
        </div>
      </DialogContainer>
      <ConfirmPopover
        isOpen={isConfirmPopoverOpen}
        onOpenChange={setIsConfirmPopoverOpen}
        onConfirm={performReroll}
        triggerRef={rerollButtonRef}
        title="Confirm key reroll"
        description="This will generate a new key. Services using the current key will need to be updated."
        confirmButtonText="Reroll key"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
