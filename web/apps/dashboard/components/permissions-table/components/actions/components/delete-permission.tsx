import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import type { Permission } from "@/lib/trpc/routers/authorization/permissions/query";
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
  Separator,
} from "@unkey/ui";
import { useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { useDeletePermission } from "./hooks/use-delete-permission";
import { PermissionInfo } from "./permission-info";

const deletePermissionFormSchema = z.object({
  confirmDeletion: z.boolean().refine((val) => val === true, {
    error: "Please confirm that you want to permanently delete this permission",
  }),
});

type DeletePermissionFormValues = z.infer<typeof deletePermissionFormSchema>;

type DeletePermissionProps = {
  permissionDetails: Permission;
} & ActionComponentProps;

export const DeletePermission = ({ permissionDetails, isOpen, onClose }: DeletePermissionProps) => {
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const methods = useForm<DeletePermissionFormValues>({
    resolver: zodResolver(deletePermissionFormSchema),
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

  const deletePermission = useDeletePermission((remainingPermissionIds) => {
    if (remainingPermissionIds.length === 0) {
      onClose();
    }
  });

  const handleDialogOpenChange = (open: boolean) => {
    if (!open && !isConfirmPopoverOpen) {
      onClose();
    }
  };

  const handleDeleteButtonClick = () => {
    setIsConfirmPopoverOpen(true);
  };

  const performPermissionDeletion = async () => {
    try {
      setIsLoading(true);
      await deletePermission.mutateAsync([permissionDetails.permissionId]);
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <FormProvider {...methods}>
        <form id="delete-permission-form">
          <DialogContainer
            isOpen={isOpen}
            subTitle="Permanently remove this permission and its assignments"
            onOpenChange={handleDialogOpenChange}
            title="Delete permission"
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="button"
                  form="delete-permission-form"
                  variant="primary"
                  color="danger"
                  size="xlg"
                  className="w-full rounded-lg"
                  disabled={!confirmDeletion || isLoading}
                  loading={isLoading}
                  onClick={handleDeleteButtonClick}
                  ref={deleteButtonRef}
                >
                  Delete permission
                </Button>
                <div className="text-gray-9 text-xs">
                  Changes may take up to 60s to propagate globally
                </div>
              </div>
            }
          >
            <PermissionInfo permissionDetails={permissionDetails} />
            <Separator className="my-3" />
            <AlertBanner variant="error">
              <IconTriangleWarningOutline12 className="size-3.5" aria-hidden="true" />
              <AlertBannerTitle>Warning</AlertBannerTitle>
              <AlertBannerDescription>
                Deleting this permission will detach it from all assigned keys and roles and remove
                its configuration. This action cannot be undone. The keys and roles themselves will
                remain available, but any usage history or references to this permission will be
                permanently lost.
              </AlertBannerDescription>
            </AlertBanner>
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
                  label="I understand this will permanently delete the permission and detach it from all assigned keys and roles"
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
        onConfirm={performPermissionDeletion}
        triggerRef={deleteButtonRef}
        title="Confirm permission deletion"
        description="This action is irreversible. All keys and roles assigned to this permission will be permanently detached."
        confirmButtonText="Delete permission"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
