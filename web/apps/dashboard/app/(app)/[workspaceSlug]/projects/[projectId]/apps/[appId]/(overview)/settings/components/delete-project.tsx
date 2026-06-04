"use client";

import { collection } from "@/lib/collections";
import { useWorkspace } from "@/providers/workspace-provider";
import { zodResolver } from "@hookform/resolvers/zod";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, DialogContainer, Input, SettingsZoneRow } from "@unkey/ui";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useProjectData } from "../../data-provider";

export function DeleteProject() {
  const { projectId, project } = useProjectData();
  const { workspace } = useWorkspace();
  const router = useRouter();
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const projectName = project?.name ?? "";

  const formSchema = z.object({
    name: z.string().refine((v) => v === projectName, "Please confirm the project name"),
  });

  type FormValues = z.infer<typeof formSchema>;

  const {
    register,
    watch,
    handleSubmit,
    formState: { isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
    defaultValues: {
      name: "",
    },
  });

  const isValid = watch("name") === projectName;

  const onSubmit = (_values: FormValues) => {
    collection.projects.delete(projectId);
    setIsDialogOpen(false);
    router.push(`/${workspace?.slug}/projects`);
  };

  return (
    <>
      <SettingsZoneRow
        title="Delete this project"
        description="Once you delete a project, there is no going back. Please be certain."
        action={{
          label: "Delete this project",
          onClick: () => setIsDialogOpen(true),
        }}
      />

      <DialogContainer
        isOpen={isDialogOpen}
        subTitle="Permanently remove this project and all associated data"
        onOpenChange={setIsDialogOpen}
        title="Delete project"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form="delete-project-form"
              variant="primary"
              color="danger"
              size="xlg"
              className="w-full rounded-lg"
              disabled={!isValid || isSubmitting}
              loading={isSubmitting}
            >
              Delete project
            </Button>
            <div className="text-gray-9 text-xs">
              This action cannot be undone – proceed with caution
            </div>
          </div>
        }
      >
        <div className="rounded-xl bg-errorA-2 dark:bg-black border border-errorA-3 flex items-center gap-4 px-[22px] py-6">
          <div className="bg-error-9 size-8 rounded-full flex items-center justify-center shrink-0">
            <TriangleWarning2 iconSize="sm-regular" className="text-white" />
          </div>
          <div className="text-error-12 text-[13px] leading-6">
            <span className="font-medium">Warning:</span> deleting{" "}
            <span className="font-medium">{projectName}</span> will remove all deployments,
            environments, custom domains, and associated data. This action cannot be undone. Any
            monitoring, logs, and historical data tied to this project will be permanently lost.
          </div>
        </div>
        <form id="delete-project-form" onSubmit={handleSubmit(onSubmit)}>
          <div className="flex flex-col gap-1 mt-4">
            <p className="text-gray-11 text-[13px]">
              Type <span className="text-gray-12 font-medium">{projectName}</span> to confirm
            </p>
            <Input {...register("name")} placeholder={`Enter "${projectName}" to confirm`} />
          </div>
        </form>
      </DialogContainer>
    </>
  );
}
