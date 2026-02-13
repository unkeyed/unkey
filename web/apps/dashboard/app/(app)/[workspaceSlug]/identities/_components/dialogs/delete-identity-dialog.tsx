"use client";

import type { IdentityResponseSchema } from "@/lib/trpc/routers/identity/query";
import { zodResolver } from "@hookform/resolvers/zod";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, ConfirmPopover, DialogContainer, FormCheckbox } from "@unkey/ui";
import { useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { useDeleteIdentity } from "./hooks/use-delete-identity";
import { IdentityInfo } from "./identity-info";

const deleteIdentityFormSchema = z.object({
  confirmDeletion: z.boolean().refine((val) => val === true, {
    error: "Please confirm that you want to permanently delete this identity",
  }),
});

type DeleteIdentityFormValues = z.infer<typeof deleteIdentityFormSchema>;

type DeleteIdentityDialogProps = {
  identity: z.infer<typeof IdentityResponseSchema>;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export const DeleteIdentityDialog = ({
  identity,
  open,
  onOpenChange,
}: DeleteIdentityDialogProps) => {
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const methods = useForm<DeleteIdentityFormValues>({
    resolver: zodResolver(deleteIdentityFormSchema),
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

  const deleteIdentity = useDeleteIdentity(() => {
    onOpenChange(false);
  });

  const handleDialogOpenChange = (open: boolean) => {
    if (isConfirmPopoverOpen) {
      // If confirm popover is active don't let this trigger outer popover
      if (!open) {
        return;
      }
    } else {
      if (!open) {
        onOpenChange(false);
      }
    }
  };

  const handleDeleteButtonClick = () => {
    setIsConfirmPopoverOpen(true);
  };

  const performIdentityDeletion = async () => {
    try {
      setIsLoading(true);
      await deleteIdentity.mutateAsync({
        identityId: identity.id,
      });
    } catch {
      // `useDeleteIdentity` already shows a toast, but we still need to
      // prevent unhandled‐rejection noise in the console.
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <FormProvider {...methods}>
        <form id="delete-identity-form">
          <DialogContainer
            isOpen={open}
            subTitle="Permanently remove this identity and its data"
            onOpenChange={handleDialogOpenChange}
            title="Delete identity"
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="button"
                  form="delete-identity-form"
                  variant="primary"
                  color="danger"
                  size="xlg"
                  className="w-full rounded-lg"
                  disabled={!confirmDeletion || isLoading}
                  loading={isLoading}
                  onClick={handleDeleteButtonClick}
                  ref={deleteButtonRef}
                >
                  Delete identity
                </Button>
                <div className="text-gray-9 text-xs">
                  Changes may take up to 60s to propagate globally
                </div>
              </div>
            }
          >
            <IdentityInfo identity={identity} />
            <div className="py-1 my-2">
              <div className="h-px bg-grayA-3 w-full" />
            </div>
            <div className="rounded-xl bg-errorA-2 dark:bg-black border border-errorA-3 flex items-center gap-4 px-[22px] py-6">
              <div className="bg-error-9 size-8 rounded-full flex items-center justify-center shrink-0">
                <TriangleWarning2 iconSize="sm-regular" className="text-white" />
              </div>
              <div className="text-error-12 text-[13px] leading-6">
                <span className="font-medium">Warning:</span> deleting this identity will remove all
                associated metadata and ratelimits. This action cannot be undone. Associated keys
                will not be affected.
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
                  label="I understand this will permanently delete the identity and all its associated data"
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
        onConfirm={performIdentityDeletion}
        triggerRef={deleteButtonRef}
        title="Confirm identity deletion"
        description="This action is irreversible. All data associated with this identity will be permanently deleted."
        confirmButtonText="Delete identity"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
