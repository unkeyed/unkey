import { ConfirmPopover } from "@/components/confirmation-popover";
import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import type { RootKey } from "@/lib/trpc/routers/settings/root-keys/query";
import { zodResolver } from "@hookform/resolvers/zod";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, DialogContainer, FormCheckbox } from "@unkey/ui";
import { useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { useDeleteRootKey } from "./hooks/use-delete-root-key";
import { RootKeyInfo } from "./root-key-info";

const deleteRootKeyFormSchema = z.object({
  confirmDeletion: z.boolean().refine((val) => val === true, {
    message: "Please confirm that you want to permanently revoke this root key",
  }),
});

type DeleteRootKeyFormValues = z.infer<typeof deleteRootKeyFormSchema>;

type DeleteRootKeyProps = { rootKeyDetails: RootKey } & ActionComponentProps;

export const DeleteRootKey = ({ rootKeyDetails, isOpen, onClose }: DeleteRootKeyProps) => {
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const methods = useForm<DeleteRootKeyFormValues>({
    resolver: zodResolver(deleteRootKeyFormSchema),
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

  const deleteRootKey = useDeleteRootKey(() => {
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

  const performRootKeyDeletion = async () => {
    try {
      setIsLoading(true);
      await deleteRootKey.mutateAsync({
        keyIds: [rootKeyDetails.id],
      });
    } catch {
      // `useDeleteRootKey` already shows a toast, but we still need to
      // prevent unhandled‐rejection noise in the console.
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <FormProvider {...methods}>
        <form id="delete-root-key-form">
          <DialogContainer
            isOpen={isOpen}
            subTitle="Delete the key permanently"
            onOpenChange={handleDialogOpenChange}
            title="Revoke Root Key"
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="button"
                  form="delete-root-key-form"
                  variant="primary"
                  color="danger"
                  size="xlg"
                  className="w-full rounded-lg"
                  disabled={!confirmDeletion || isLoading}
                  loading={isLoading}
                  onClick={handleDeleteButtonClick}
                  ref={deleteButtonRef}
                >
                  Delete permanently
                </Button>
                <div className="text-gray-9 text-xs">
                  Changes may take up to 60s to propagate globally
                </div>
              </div>
            }
          >
            <RootKeyInfo rootKeyDetails={rootKeyDetails} />
            <div className="py-1 my-2">
              <div className="h-[1px] bg-grayA-3 w-full" />
            </div>
            <div className="rounded-xl bg-errorA-2 dark:bg-black border border-errorA-3 flex items-center gap-4 px-[22px] py-6">
              <div className="bg-error-9 size-8 rounded-full flex items-center justify-center flex-shrink-0">
                <TriangleWarning2 iconsize="sm-regular" className="text-white" />
              </div>
              <div className="text-error-12 text-[13px] leading-6">
                <span className="font-medium">Warning:</span> This action can not be undone. Your
                root key will no longer be able to create resources.
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
                  size="md"
                  checked={field.value}
                  onCheckedChange={field.onChange}
                  label="I understand this will permanently delete the root key."
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
        onConfirm={performRootKeyDeletion}
        triggerRef={deleteButtonRef}
        title="Confirm root key deletion"
        description="This action is irreversible. The root key will be permanently removed and will no longer be able to create resources."
        confirmButtonText="Delete permanently"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
