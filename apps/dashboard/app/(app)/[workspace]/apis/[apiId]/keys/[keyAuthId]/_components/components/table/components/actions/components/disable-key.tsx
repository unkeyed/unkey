import { revalidate } from "@/app/actions";
import { ConfirmPopover } from "@/components/confirmation-popover";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, DialogContainer, FormCheckbox } from "@unkey/ui";
import { useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { useUpdateKeyStatus } from "./hooks/use-update-key-status";
import { KeyInfo } from "./key-info";

const updateKeyStatusFormSchema = z.object({
  confirmStatusChange: z.boolean().refine((val) => val === true, {
    message: "Please confirm that you want to change this key's status",
  }),
});

type UpdateKeyStatusFormValues = z.infer<typeof updateKeyStatusFormSchema>;

type UpdateKeyStatusProps = { keyDetails: KeyDetails } & ActionComponentProps;

export const UpdateKeyStatus = ({ keyDetails, isOpen, onClose }: UpdateKeyStatusProps) => {
  const isEnabling = !keyDetails.enabled;
  const action = isEnabling ? "Enable" : "Disable";
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const actionButtonRef = useRef<HTMLButtonElement>(null);

  const methods = useForm<UpdateKeyStatusFormValues>({
    resolver: zodResolver(updateKeyStatusFormSchema),
    mode: "onChange",
    shouldFocusError: true,
    shouldUnregister: true,
    defaultValues: {
      confirmStatusChange: false,
    },
  });

  const {
    formState: { errors },
    control,
    watch,
  } = methods;

  const confirmStatusChange = watch("confirmStatusChange");

  const updateKeyStatus = useUpdateKeyStatus(() => {
    onClose();
    revalidate(keyDetails.id);
  });

  const handleDialogOpenChange = (open: boolean) => {
    if (isConfirmPopoverOpen) {
      // If confirm popover is active don't let this trigger outer popover
      if (!open) {
        return;
      }
    } else {
      if (!open) {
        onClose();
      }
    }
  };

  const handleActionButtonClick = () => {
    // Only show confirmation popover for disabling
    if (isEnabling) {
      performStatusUpdate();
    } else {
      setIsConfirmPopoverOpen(true);
    }
  };

  const performStatusUpdate = async () => {
    try {
      setIsLoading(true);
      await updateKeyStatus.mutateAsync({
        keyIds: [keyDetails.id],
        enabled: isEnabling,
      });
    } catch {
      // `useUpdateKeyStatus` already shows a toast, but we still need to
      // prevent unhandled‐rejection noise in the console.
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <FormProvider {...methods}>
        <form id="update-key-status-form">
          <DialogContainer
            isOpen={isOpen}
            subTitle={
              isEnabling
                ? "Enable this key to allow verification requests"
                : "Disable this key to block verification requests"
            }
            onOpenChange={handleDialogOpenChange}
            title={`${action} key`}
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="button"
                  form="update-key-status-form"
                  variant="primary"
                  color={isEnabling ? "default" : "danger"}
                  size="xlg"
                  className="w-full rounded-lg"
                  disabled={!confirmStatusChange || isLoading}
                  loading={isLoading}
                  onClick={handleActionButtonClick}
                  ref={actionButtonRef}
                >
                  {`${action} key`}
                </Button>
                <div className="text-gray-9 text-xs">Changes will be applied immediately</div>
              </div>
            }
          >
            <KeyInfo keyDetails={keyDetails} />
            <div className="py-1 my-2">
              <div className="h-[1px] bg-grayA-3 w-full" />
            </div>
            <Controller
              name="confirmStatusChange"
              control={control}
              render={({ field }) => (
                <FormCheckbox
                  id="confirm-status-change"
                  color={isEnabling ? "default" : "danger"}
                  size="lg"
                  onCheckedChange={field.onChange}
                  required
                  label={
                    isEnabling
                      ? "I want to enable this key and allow verification"
                      : "I want to disable this key and stop all verification"
                  }
                  error={errors.confirmStatusChange?.message}
                />
              )}
            />
          </DialogContainer>
        </form>
      </FormProvider>
      {!isEnabling && (
        <ConfirmPopover
          isOpen={isConfirmPopoverOpen}
          onOpenChange={setIsConfirmPopoverOpen}
          onConfirm={performStatusUpdate}
          triggerRef={actionButtonRef}
          title="Confirm disabling key"
          description="This will disable the key and prevent any verification requests from being processed."
          confirmButtonText="Disable key"
          cancelButtonText="Cancel"
          variant="danger"
        />
      )}
    </>
  );
};
