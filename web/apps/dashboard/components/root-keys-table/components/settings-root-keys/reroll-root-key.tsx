import { SecretKey } from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_components/create-key/components/secret-key";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { CircleInfo, Refresh3 } from "@unkey/icons";
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
  VisuallyHidden,
} from "@unkey/ui";
import { useRef, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { useRerollRootKey } from "../../hooks/use-reroll-root-key";
import { RootKeyInfo } from "./root-key-info";

const EXPIRATION_OPTIONS = [
  { label: "Immediately", value: "0" },
  { label: "1 hour", value: "3600000" },
  { label: "24 hours", value: "86400000" },
  { label: "7 days", value: "604800000" },
  { label: "30 days", value: "2592000000" },
] as const;

type RerollRootKeyProps = { rootKeyDetails: RootKey } & ActionComponentProps;

export const RerollRootKey = ({ rootKeyDetails, isOpen, onClose }: RerollRootKeyProps) => {
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

  const rerollRootKey = useRerollRootKey((data) => {
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
      await rerollRootKey.mutateAsync({
        keyId: rootKeyDetails.id,
        expiration: Number.parseInt(expiration, 10),
      });
    } catch {
      // useRerollRootKey already shows a toast
    } finally {
      setIsLoading(false);
    }
  };

  const handleSuccessClose = () => {
    setIsSuccessConfirmOpen(false);
    setNewKeyData(null);
    rerollRootKey.invalidateKeys();
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
          className="drop-shadow-2xl transform-gpu border-grayA-4 overflow-hidden rounded-2xl! p-0 gap-0 w-full max-w-[760px] max-h-[90vh] overflow-y-auto"
          showCloseWarning
          onAttemptClose={() => setIsSuccessConfirmOpen(true)}
        >
          <VisuallyHidden asChild>
            <DialogTitle>Root Key Rerolled Successfully</DialogTitle>
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
                  <Refresh3 iconSize="2xl-thin" aria-hidden="true" focusable={false} />
                  <div className="flex items-center justify-center border border-grayA-3 rounded-full bg-success-9 text-white size-[22px] absolute right-[-10px] top-[-10px]">
                    <svg
                      width="12"
                      height="12"
                      viewBox="0 0 12 12"
                      fill="none"
                      xmlns="http://www.w3.org/2000/svg"
                      aria-hidden="true"
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
                <div className="border border-grayA-4 rounded-[10px] size-14" />
                <div className="border border-grayA-4 rounded-[10px] size-14 opacity-35" />
              </div>
            </div>
            <div className="mt-5 flex flex-col gap-2 items-center">
              <div className="font-semibold text-gray-12 text-[16px] leading-[24px]">
                Root key rerolled
              </div>
              <div
                className="text-gray-10 text-[13px] leading-[24px] text-center"
                ref={successDividerRef}
              >
                Your root key has been successfully rerolled.
                <br /> The old key will{" "}
                {expiration === "0"
                  ? "be revoked immediately"
                  : `expire in ${EXPIRATION_OPTIONS.find((o) => o.value === expiration)?.label?.toLowerCase()}`}
                .
              </div>
            </div>
            <div className="p-1 w-full my-8">
              <div className="h-px bg-grayA-3 w-full" />
            </div>
            <div className="flex flex-col gap-2 items-start w-full">
              <div className="text-gray-12 text-sm font-semibold">Root Key</div>
              <SecretKey
                value={newKeyData.key}
                title="Root Key"
                className="bg-white dark:bg-black"
              />
              <div className="text-gray-9 text-[13px] flex items-center gap-1.5">
                <CircleInfo
                  className="text-accent-9"
                  iconSize="sm-regular"
                  aria-hidden="true"
                  focusable={false}
                />
                <span className="flex items-center gap-1">
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
            <div className="mt-6">
              <div className="mt-4 text-center text-gray-10 text-xs leading-6">
                All set! Your new root key has the same permissions as the original.
              </div>
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
            }}
          />
        </DialogContent>
      </Dialog>
    );
  }

  return (
    <>
      <DialogContainer
        isOpen={isOpen}
        subTitle="Generate a new root key while preserving all permissions"
        onOpenChange={handleDialogOpenChange}
        title="Reroll root key"
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
              Reroll root key
            </Button>
            <div className="text-gray-9 text-xs">
              Changes may take up to 60s to propagate globally
            </div>
          </div>
        }
      >
        <RootKeyInfo rootKeyDetails={rootKeyDetails} />
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
            How long the original root key remains valid after rerolling
          </p>
        </div>
      </DialogContainer>
      <ConfirmPopover
        isOpen={isConfirmPopoverOpen}
        onOpenChange={setIsConfirmPopoverOpen}
        onConfirm={performReroll}
        triggerRef={rerollButtonRef}
        title="Confirm root key reroll"
        description="This will generate a new root key. Services using the current key will need to be updated."
        confirmButtonText="Reroll root key"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
