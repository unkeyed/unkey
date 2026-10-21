"use client";

import { EnvironmentSelect } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-select";
import type { App } from "@/lib/collections/deploy/apps";
import { sanitizeImageRef, validateImageRef } from "@/lib/docker-image-ref";
import { zodResolver } from "@hookform/resolvers/zod";
import { match } from "@unkey/match";
import { FormDescription, FormInput, FormLabel, InfoTooltip, TimestampInfo } from "@unkey/ui";
import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { useProjectData } from "../data-provider";
import { useRecentDeployments } from "../hooks/use-recent-deployments";
import { DeploymentDialog } from "./deployment-dialog";
import { useCreateDeployment } from "./use-create-deployment";

const MAX_IMAGE_ROWS = 10;

const DEPLOYED_STATUSES = new Set(["ready", "stopped", "superseded"]);

const imageFormSchema = z.object({
  environmentId: z.string().min(1, "Environment is required"),
  image: z
    .string()
    .trim()
    .superRefine((value, ctx) => {
      const result = validateImageRef(sanitizeImageRef(value));
      if (!result.ok) {
        ctx.addIssue({ code: "custom", message: result.error });
      }
    }),
});

type ImageFormValues = z.infer<typeof imageFormSchema>;

type ImageDeploymentFormProps = {
  app: App;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
};

export function ImageDeploymentForm({ app, isOpen, onOpenChange }: ImageDeploymentFormProps) {
  const { environments } = useProjectData();
  const { deployments } = useRecentDeployments({ enabled: isOpen });
  const defaultEnvironmentId =
    environments.find((env) => env.kind === "preview")?.id ?? environments[0]?.id ?? "";

  const {
    register,
    handleSubmit,
    setValue,
    reset,
    watch,
    getFieldState,
    control,
    formState: { errors, isValid, isSubmitting },
  } = useForm<ImageFormValues>({
    resolver: zodResolver(imageFormSchema),
    mode: "onChange",
    defaultValues: { environmentId: defaultEnvironmentId },
  });

  const close = () => {
    onOpenChange(false);
    reset({ environmentId: defaultEnvironmentId });
  };
  const createDeployment = useCreateDeployment(close);

  const imageRef = sanitizeImageRef(watch("image") ?? "");
  const imageValidation = validateImageRef(imageRef);

  useEffect(() => {
    if (defaultEnvironmentId) {
      setValue("environmentId", defaultEnvironmentId, { shouldValidate: true });
    }
  }, [defaultEnvironmentId, setValue]);

  useEffect(() => {
    if (isOpen && app.imageReference && !getFieldState("image").isDirty) {
      setValue("image", app.imageReference, { shouldValidate: true });
    }
  }, [app.imageReference, getFieldState, isOpen, setValue]);

  function onSubmit(values: ImageFormValues) {
    const environment = environments.find((env) => env.id === values.environmentId);
    if (environment) {
      createDeployment.mutate({ environment: environment.slug, oci: imageRef });
    }
  }

  // Past successfully deployed prebuilt images, deduped by image ref
  // (deployments are ordered newest-first, so the latest deployment wins).
  const imageRows = deployments
    .filter(
      (d, i, all) =>
        match(d.source)
          .with("oci", () => true)
          .with("git", "unknown", () => false)
          .exhaustive() &&
        d.resolvedImage &&
        DEPLOYED_STATUSES.has(d.status) &&
        all.findIndex((o) => o.resolvedImage === d.resolvedImage) === i,
    )
    .slice(0, MAX_IMAGE_ROWS);

  return (
    <DeploymentDialog
      isOpen={isOpen}
      onOpenChange={(open) => (open ? onOpenChange(true) : close())}
      subTitle="Deploy a prebuilt image or redeploy a previous one"
      submitLabel="Deploy"
      canSubmit={isValid}
      isSubmitting={createDeployment.isLoading || isSubmitting}
      onSubmit={handleSubmit(onSubmit)}
    >
      <fieldset className="flex flex-col gap-2 border-0 m-0 p-0">
        <FormLabel label="Environment" htmlFor="environment-select" />
        <Controller
          control={control}
          name="environmentId"
          render={({ field }) => (
            <EnvironmentSelect
              id="environment-select"
              value={field.value}
              onValueChange={field.onChange}
              includeAll={false}
              placeholder="Select environment"
              variant={errors.environmentId ? "error" : "default"}
            />
          )}
        />
        <FormDescription
          description="Target environment for this deployment."
          descriptionId="environment-select-description"
          errorId="environment-select-error"
          error={errors.environmentId?.message}
        />
      </fieldset>
      <div className="flex flex-col gap-2">
        <FormInput
          label="Image Reference"
          className="min-h-9"
          autoComplete="off"
          data-1p-ignore
          description="Paste a container image reference to deploy, or pick a previously deployed image below."
          error={errors.image?.message}
          {...register("image", {
            onBlur: () => setValue("image", imageRef, { shouldValidate: true }),
          })}
          onPaste={(e) => {
            const cleaned = sanitizeImageRef(e.clipboardData.getData("text"));
            e.preventDefault();
            setValue("image", cleaned, { shouldValidate: true });
          }}
          placeholder="ghcr.io/unkeyed/deploy-demo:latest"
        />
        {imageValidation.ok && imageValidation.warning ? (
          <output className="text-warning-11 text-sm">{imageValidation.warning}</output>
        ) : null}
      </div>
      {imageRows.length > 0 && (
        <div className="flex flex-col divide-y divide-gray-4 rounded-md border overflow-hidden">
          {imageRows.map((deployment) => (
            <div
              key={deployment.id}
              className="relative flex items-center justify-between px-3 py-2 bg-grayA-2 hover:bg-grayA-3 transition-colors text-sm text-grayA-11"
            >
              <InfoTooltip
                content={deployment.resolvedImage}
                asChild
                position={{ align: "start", side: "top" }}
              >
                <button
                  type="button"
                  aria-label={`Select ${deployment.resolvedImage}`}
                  onClick={() =>
                    setValue("image", deployment.resolvedImage ?? "", {
                      shouldValidate: true,
                      shouldDirty: true,
                    })
                  }
                  className="absolute inset-0 z-10 cursor-pointer focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-inset focus-visible:ring-grayA-7"
                />
              </InfoTooltip>
              <span className="min-w-0 max-w-[300px] truncate">{deployment.resolvedImage}</span>
              <TimestampInfo
                value={deployment.createdAt}
                displayType="relative"
                className="relative z-20 text-gray-11 shrink-0 ml-3"
              />
            </div>
          ))}
        </div>
      )}
    </DeploymentDialog>
  );
}
