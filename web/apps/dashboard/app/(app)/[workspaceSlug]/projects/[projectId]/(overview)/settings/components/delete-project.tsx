"use client";

import { collection } from "@/lib/collections";
import { useWorkspace } from "@/providers/workspace-provider";
import { zodResolver } from "@hookform/resolvers/zod";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, DialogContainer, Input } from "@unkey/ui";
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
      <div className="w-225 mx-auto mt-10 mb-14">
        <h2 className="font-semibold text-error-11 text-lg mb-4">Danger Zone</h2>
        <div className="rounded-lg border border-error-7 overflow-hidden">
          <div className="flex items-center justify-between px-6 py-5">
            <div>
              <p className="font-medium text-gray-12 text-sm">Delete this project</p>
              <p className="text-gray-11 text-[13px] mt-0.5">
                Once you delete a project, there is no going back. Please be certain.
              </p>
            </div>
            <Button
              variant="destructive"
              size="lg"
              className="shrink-0"
              onClick={() => setIsDialogOpen(true)}
            >
              Delete this project
            </Button>
          </div>
        </div>
      </div>

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
