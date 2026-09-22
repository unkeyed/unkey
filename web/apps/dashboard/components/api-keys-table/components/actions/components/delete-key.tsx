import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { RecentlyUsedKeyWarning } from "@/components/recently-used-key-warning";
import { RECENTLY_USED_WINDOW_LABEL, isRecentlyUsed } from "@/lib/recently-used-key";
import type { KeyDetails } from "@/lib/trpc/routers/api/keys/query-api-keys/schema";
import { zodResolver } from "@hookform/resolvers/zod";
import { IconTriangleWarningOutline12 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerDescription,
  AlertBannerTitle,
  Button,
  ConfirmPopover,
  DialogContainer,
  FormCheckbox,
} from "@unkey/ui";
import { useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { useDeleteKey } from "./hooks/use-delete-key";
import { KeyInfo } from "./key-info";

const deleteKeyFormSchema = z.object({
  confirmDeletion: z.boolean().refine((val) => val === true, {
    error: "Please confirm that you want to permanently delete this key",
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
  const recentlyUsed = isRecentlyUsed(keyDetails.last_used_at);

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
              <div className="h-px bg-grayA-3 w-full" />
            </div>
            <AlertBanner variant="error">
              <IconTriangleWarningOutline12 className="size-3.5" aria-hidden="true" />
              <AlertBannerTitle>Warning:</AlertBannerTitle>
              <AlertBannerDescription>
                deleting this key will remove all associated data and metadata. This action cannot
                be undone.
              </AlertBannerDescription>
            </AlertBanner>
            {recentlyUsed && <RecentlyUsedKeyWarning lastUsedAt={keyDetails.last_used_at} />}
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
                  label="I understand this will permanently delete the key and its associated metadata"
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
        description={
          recentlyUsed
            ? `This key was used in the last ${RECENTLY_USED_WINDOW_LABEL} and may still be live. This action is irreversible. Metadata and ratelimits associated with this key will be permanently deleted.`
            : "This action is irreversible. Metadata and ratelimits associated with this key will be permanently deleted."
        }
        confirmButtonText="Delete key"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
