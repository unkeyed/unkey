import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, ConfirmPopover, DialogContainer, FormCheckbox } from "@unkey/ui";
import { useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { useDeleteKey } from "./hooks/use-delete-key";
import { KeyInfo } from "./key-info";

const deleteKeyFormSchema = z.object({
  confirmDeletion: z.boolean().refine((val) => val === true, {
    message: "Please confirm that you want to permanently delete this key",
  }),
});

type DeleteKeyFormValues = z.infer<typeof deleteKeyFormSchema>;

type DeleteKeyProps = { keyDetails: KeyDetails } & ActionComponentProps;

export const DeleteKey = ({ keyDetails, isOpen, onClose }: DeleteKeyProps) => {
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const methods = useForm<DeleteKeyFormValues>({
    resolver: zodResolver(deleteKeyFormSchema),
    mode: "onChange",
    shouldFocusError: true,
    shouldUnregister: true,
    defaultValues: {
      confirmDeletion: false,
    },
  });

  const {
    formState: { errors },
    control,
    watch,
  } = methods;

  const confirmDeletion = watch("confirmDeletion");

  const deleteKey = useDeleteKey(() => {
    onClose();
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

  const handleDeleteButtonClick = () => {
    setIsConfirmPopoverOpen(true);
  };

  const performKeyDeletion = async () => {
    try {
      setIsLoading(true);
      await deleteKey.mutateAsync({
        keyIds: [keyDetails.id],
      });
    } catch {
      // `useDeleteKey` already shows a toast, but we still need to
      // prevent unhandled‐rejection noise in the console.
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <FormProvider {...methods}>
        <form id="delete-key-form">
          <DialogContainer
            isOpen={isOpen}
            subTitle="Permanently remove this key and its data"
            onOpenChange={handleDialogOpenChange}
            title="Delete key"
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="button"
                  form="delete-key-form"
                  variant="primary"
                  color="danger"
                  size="xlg"
                  className="w-full rounded-lg"
                  disabled={!confirmDeletion || isLoading}
                  loading={isLoading}
                  onClick={handleDeleteButtonClick}
                  ref={deleteButtonRef}
                >
                  Delete key
                </Button>
                <div className="text-gray-9 text-xs">
                  This key will be permanently deleted immediately
                </div>
              </div>
            }
          >
            <KeyInfo keyDetails={keyDetails} />
            <div className="py-1 my-2">
              <div className="h-[1px] bg-grayA-3 w-full" />
            </div>
            <div className="rounded-xl bg-errorA-2 dark:bg-black border border-errorA-3 flex items-center gap-4 px-[22px] py-6">
              <div className="bg-error-9 size-8 rounded-full flex items-center justify-center flex-shrink-0">
                <TriangleWarning2 iconSize="sm-regular" className="text-white" />
              </div>
              <div className="text-error-12 text-[13px] leading-6">
                <span className="font-medium">Warning:</span> deleting this key will remove all
                associated data and metadata. This action cannot be undone. Any verification,
                tracking, and historical usage tied to this key will be permanently lost.
              </div>
            </div>
            <Controller
              name="confirmDeletion"
              control={control}
              render={({ field }) => (
                <FormCheckbox
                  id="confirm-deletion"
                  className="mt-2"
                  color="danger"
                  size="lg"
                  checked={field.value}
                  onCheckedChange={field.onChange}
                  label="I understand this will permanently delete the key and all its associated data"
                  error={errors.confirmDeletion?.message}
                />
              )}
            />
          </DialogContainer>
        </form>
      </FormProvider>
      <ConfirmPopover
        isOpen={isConfirmPopoverOpen}
        onOpenChange={setIsConfirmPopoverOpen}
        onConfirm={performKeyDeletion}
        triggerRef={deleteButtonRef}
        title="Confirm key deletion"
        description="This action is irreversible. All data associated with this key will be permanently deleted."
        confirmButtonText="Delete key"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
