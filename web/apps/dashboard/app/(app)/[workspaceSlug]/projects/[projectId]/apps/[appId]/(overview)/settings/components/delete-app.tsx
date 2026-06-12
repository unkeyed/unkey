"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { zodResolver } from "@hookform/resolvers/zod";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { TriangleWarning2 } from "@unkey/icons";
import { Button, DialogContainer, Input, SettingsZoneRow } from "@unkey/ui";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useAppId, useProjectData } from "../../data-provider";

export function DeleteApp() {
  const { projectId } = useProjectData();
  const appId = useAppId();
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const [isDialogOpen, setIsDialogOpen] = useState(false);

  const appsQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const appName = appsQuery.data?.[0]?.name ?? "";

  const formSchema = z.object({
    name: z.string().refine((v) => v === appName, "Please confirm the app name"),
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

  const isValid = watch("name") === appName;

  const onSubmit = (_values: FormValues) => {
    collection.apps.delete(appId);
    setIsDialogOpen(false);
    router.push(routes.projects.detail({ workspaceSlug: workspace.slug, projectId }));
  };

  return (
    <>
      <SettingsZoneRow
        title="Delete this app"
        description="Once you delete an app, there is no going back. Please be certain."
        action={{
          label: "Delete this app",
          onClick: () => setIsDialogOpen(true),
        }}
      />

      <DialogContainer
        isOpen={isDialogOpen}
        subTitle="Permanently remove this app and all associated data"
        onOpenChange={setIsDialogOpen}
        title="Delete app"
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form="delete-app-form"
              variant="primary"
              color="danger"
              size="xlg"
              className="w-full rounded-lg"
              disabled={!isValid || isSubmitting}
              loading={isSubmitting}
            >
              Delete app
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
            <span className="font-medium">{appName}</span> will remove all deployments,
            environments, custom domains, and associated data. This action cannot be undone. Any
            monitoring, logs, and historical data tied to this app will be permanently lost.
          </div>
        </div>
        <form id="delete-app-form" onSubmit={handleSubmit(onSubmit)}>
          <div className="flex flex-col gap-1 mt-4">
            <p className="text-gray-11 text-[13px]">
              Type <span className="text-gray-12 font-medium">{appName}</span> to confirm
            </p>
            <Input {...register("name")} placeholder={`Enter "${appName}" to confirm`} />
          </div>
        </form>
      </DialogContainer>
    </>
  );
}
