"use client";

import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import { routes } from "@/lib/navigation/routes";
import { zodResolver } from "@hookform/resolvers/zod";
import { IconTriangleWarningOutline12 } from "@unkey/icons";
import {
  AlertBanner,
  AlertBannerDescription,
  Button,
  DialogContainer,
  Input,
  SettingsZoneRow,
} from "@unkey/ui";

import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useApp } from "../../../../_components/settings/hooks/use-app";

export function DeleteApp() {
  const { projectId, appId, app } = useApp();
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const [isDialogOpen, setIsDialogOpen] = useState(false);
  const appName = app?.name ?? "";

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
        description="This cannot be undone."
        action={{
          label: "Delete this app",
          onClick: () => setIsDialogOpen(true),
        }}
      />

      <DialogContainer
        isOpen={isDialogOpen}
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
          </div>
        }
      >
        <AlertBanner variant="error">
          <IconTriangleWarningOutline12 aria-hidden="true" />
          <AlertBannerDescription>
            Deleting <span className="font-medium">{appName}</span> removes its deployments,
            environments, custom domains, logs and history. This cannot be undone.
          </AlertBannerDescription>
        </AlertBanner>
        <form id="delete-app-form" onSubmit={handleSubmit(onSubmit)}>
          <div className="flex flex-col gap-1 mt-4">
            <p className="text-gray-11 text-sm">
              Type <span className="text-gray-12 font-medium">{appName}</span> to confirm
            </p>
            <Input {...register("name")} />
          </div>
        </form>
      </DialogContainer>
    </>
  );
}
