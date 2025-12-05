import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import type { RoleBasic } from "@/lib/trpc/routers/authorization/roles/query";
import { zodResolver } from "@hookform/resolvers/zod";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, ConfirmPopover, DialogContainer, FormCheckbox } from "@unkey/ui";
import { useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { useDeleteRole } from "./hooks/use-delete-role";
import { RoleInfo } from "./role-info";

const deleteRoleFormSchema = z.object({
  confirmDeletion: z.boolean().refine((val) => val === true, {
    message: "Please confirm that you want to permanently delete this role",
  }),
});

type DeleteRoleFormValues = z.infer<typeof deleteRoleFormSchema>;

type DeleteRoleProps = { roleDetails: RoleBasic } & ActionComponentProps;

export const DeleteRole = ({ roleDetails, isOpen, onClose }: DeleteRoleProps) => {
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const methods = useForm<DeleteRoleFormValues>({
    resolver: zodResolver(deleteRoleFormSchema),
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

  const deleteRole = useDeleteRole(() => {
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

  const performRoleDeletion = async () => {
    try {
      setIsLoading(true);
      await deleteRole.mutateAsync({
        roleIds: roleDetails.roleId,
      });
    } catch {
      // `useDeleteRole` already shows a toast, but we still need to
      // prevent unhandled‐rejection noise in the console.
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <FormProvider {...methods}>
        <form id="delete-role-form">
          <DialogContainer
            isOpen={isOpen}
            subTitle="Permanently remove this role and its assignments"
            onOpenChange={handleDialogOpenChange}
            title="Delete role"
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="button"
                  form="delete-role-form"
                  variant="primary"
                  color="danger"
                  size="xlg"
                  className="w-full rounded-lg"
                  disabled={!confirmDeletion || isLoading}
                  loading={isLoading}
                  onClick={handleDeleteButtonClick}
                  ref={deleteButtonRef}
                >
                  Delete role
                </Button>
                <div className="text-gray-9 text-xs">
                  Changes may take up to 60s to propagate globally
                </div>
              </div>
            }
          >
            <RoleInfo roleDetails={roleDetails} />
            <div className="py-1 my-2">
              <div className="h-[1px] bg-grayA-3 w-full" />
            </div>
            <div className="rounded-xl bg-errorA-2 dark:bg-black border border-errorA-3 flex items-center gap-4 px-[22px] py-6">
              <div className="bg-error-9 size-8 rounded-full flex items-center justify-center flex-shrink-0">
                <TriangleWarning2 iconSize="sm-regular" className="text-white" />
              </div>
              <div className="text-error-12 text-[13px] leading-6">
                <span className="font-medium">Warning:</span> deleting this role will detach it from
                all assigned keys and permissions and remove its configuration. This action cannot
                be undone. The permissions and keys themselves will remain available, but any usage
                history or references to this role will be permanently lost.
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
                  label="I understand this will permanently delete the role and detach it from all assigned keys and permissions"
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
        onConfirm={performRoleDeletion}
        triggerRef={deleteButtonRef}
        title="Confirm role deletion"
        description="This action is irreversible. All permissions and keys for this role will be permanently removed."
        confirmButtonText="Delete role"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};
