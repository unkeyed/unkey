"use client";

import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { collection } from "@/lib/collections";
import { zodResolver } from "@hookform/resolvers/zod";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, ConfirmPopover, DialogContainer, FormCheckbox } from "@unkey/ui";
import { useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { ProjectInfo } from "./project-info";

const deleteProjectFormSchema = z.object({
  confirmDeletion: z.boolean().refine((val) => val === true, {
    message: "Please confirm that you want to permanently delete this project",
  }),
});

type DeleteProjectFormValues = z.infer<typeof deleteProjectFormSchema>;

type Props = {
  projectId: string;
  projectName: string;
} & ActionComponentProps;

export function DeleteProjectDialog({ projectId, projectName, isOpen, onClose }: Props) {
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const methods = useForm<DeleteProjectFormValues>({
    resolver: zodResolver(deleteProjectFormSchema),
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

  const handleDialogOpenChange = (open: boolean) => {
    if (isConfirmPopoverOpen) {
      // If confirm popover is active don't let this trigger outer dialog
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

  const performProjectDeletion = () => {
    collection.projects.delete(projectId);
    onClose();
  };

  return (
    <>
      <FormProvider {...methods}>
        <form id="delete-project-form">
          <DialogContainer
            isOpen={isOpen}
            subTitle="Permanently remove this project and all associated data"
            onOpenChange={handleDialogOpenChange}
            title="Delete project"
            footer={
              <div className="w-full flex flex-col gap-2 items-center justify-center">
                <Button
                  type="button"
                  form="delete-project-form"
                  variant="primary"
                  color="danger"
                  size="xlg"
                  className="w-full rounded-lg"
                  disabled={!confirmDeletion}
                  onClick={handleDeleteButtonClick}
                  ref={deleteButtonRef}
                >
                  Delete project
                </Button>
                <div className="text-gray-9 text-xs">
                  This project will be permanently deleted immediately
                </div>
              </div>
            }
          >
            <ProjectInfo projectId={projectId} projectName={projectName} />
            <div className="py-1 my-2">
              <div className="h-[1px] bg-grayA-3 w-full" />
            </div>
            <div className="rounded-xl bg-errorA-2 dark:bg-black border border-errorA-3 flex items-center gap-4 px-[22px] py-6">
              <div className="bg-error-9 size-8 rounded-full flex items-center justify-center flex-shrink-0">
                <TriangleWarning2 iconSize="sm-regular" className="text-white" />
              </div>
              <div className="text-error-12 text-[13px] leading-6">
                <span className="font-medium">Warning:</span> deleting this project will remove all
                deployments, environments, custom domains, and associated data. This action cannot
                be undone. Any monitoring, logs, and historical data tied to this project will be
                permanently lost.
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
                  label="I understand this will permanently delete the project and all its associated data"
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
        onConfirm={performProjectDeletion}
        triggerRef={deleteButtonRef}
        title="Confirm project deletion"
        description="This action is irreversible. All data associated with this project will be permanently deleted."
        confirmButtonText="Delete project"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
}
