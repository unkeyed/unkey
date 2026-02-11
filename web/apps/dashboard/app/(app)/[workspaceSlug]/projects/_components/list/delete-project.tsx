import type { ActionComponentProps } from "@/components/logs/table-action.popover";
import { collection } from "@/lib/collections";
import { eq, useLiveQuery } from "@tanstack/react-db";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, ConfirmPopover, DialogContainer, FormCheckbox } from "@unkey/ui";
import { useRef, useState } from "react";
import { Controller, FormProvider, useForm } from "react-hook-form";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { useDeleteProject } from "./use-delete-project";

const deleteProjectFormSchema = z.object({
  confirmDeletion: z.boolean().refine((val) => val === true, {
    error: "Please confirm that you want to permanently delete this project",
  }),
});

type DeleteProjectFormValues = z.infer<typeof deleteProjectFormSchema>;

type DeleteProjectProps = { projectId: string } & ActionComponentProps;

export const DeleteProject = ({ projectId, isOpen, onClose }: DeleteProjectProps) => {
  const [isConfirmPopoverOpen, setIsConfirmPopoverOpen] = useState(false);
  const [isLoading, setIsLoading] = useState(false);
  const deleteButtonRef = useRef<HTMLButtonElement>(null);

  const project = useLiveQuery((q) =>
    q
      .from({ project: collection.projects })
      .where(({ project }) => eq(project.id, projectId))
      .select(({ project }) => ({ id: project.id, name: project.name, slug: project.slug })),
  ).data.at(0);

  const methods = useForm<DeleteProjectFormValues>({
    resolver: zodResolver(deleteProjectFormSchema),
    mode: "onChange",
    shouldFocusError: true,
    shouldUnregister: true,
    defaultValues: { confirmDeletion: false },
  });

  const {
    formState: { errors },
    control,
    watch,
  } = methods;

  const confirmDeletion = watch("confirmDeletion");

  const deleteProject = useDeleteProject(() => {
    onClose();
  });

  const handleDialogOpenChange = (open: boolean) => {
    if (isConfirmPopoverOpen) {
      if (!open) return;
    } else if (!open) {
      onClose();
    }
  };

  const handleDeleteButtonClick = () => {
    setIsConfirmPopoverOpen(true);
  };

  const performProjectDeletion = async () => {
    try {
      setIsLoading(true);
      await deleteProject.mutateAsync({ projectId });
    } catch {
      // mutation handles toast; suppress unhandled rejection noise
    } finally {
      setIsLoading(false);
    }
  };

  return (
    <>
      <FormProvider {...methods}>
        <form id="delete-project-form">
          <DialogContainer
            isOpen={isOpen}
            subTitle="Permanently remove this project and its related resources"
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
                  disabled={!confirmDeletion || isLoading}
                  loading={isLoading}
                  onClick={handleDeleteButtonClick}
                  ref={deleteButtonRef}
                >
                  Delete project
                </Button>
                <div className="text-gray-9 text-xs">This action cannot be undone</div>
              </div>
            }
          >
            <div className="text-sm text-gray-12">
              You are deleting{" "}
              <span className="font-medium">
                {project?.name ?? "this project"}
                {project?.slug ? ` (${project.slug})` : ""}
              </span>
              .
            </div>

            <div className="py-1 my-2">
              <div className="h-[1px] bg-grayA-3 w-full" />
            </div>

            <div className="rounded-xl bg-errorA-2 dark:bg-black border border-errorA-3 flex items-center gap-4 px-[22px] py-6">
              <div className="bg-error-9 size-8 rounded-full flex items-center justify-center flex-shrink-0">
                <TriangleWarning2 iconSize="sm-regular" className="text-white" />
              </div>
              <div className="text-error-12 text-[13px] leading-6">
                <span className="font-medium">Warning:</span> This will permanently delete the
                project and remove associated deployments, environments, routes, and connections.
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
                  label="I understand this will permanently delete the project."
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
        description="This action is irreversible. The project and its related resources will be permanently deleted."
        confirmButtonText="Delete project"
        cancelButtonText="Cancel"
        variant="danger"
      />
    </>
  );
};

