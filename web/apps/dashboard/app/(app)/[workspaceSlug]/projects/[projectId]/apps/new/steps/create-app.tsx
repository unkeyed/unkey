"use client";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { collection } from "@/lib/collections";
import { trpcClient } from "@/lib/collections/client";
import { createAppRequestSchema } from "@/lib/collections/deploy/apps";
import { applyDefaultSettings } from "@/lib/collections/deploy/environment-settings";
import { SERVER_PLACEHOLDER } from "@/lib/collections/deploy/utils";
import { slugify } from "@/lib/slugify";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { DuplicateKeyError } from "@tanstack/react-db";
import { Button, FormInput, toast, useStepWizard } from "@unkey/ui";
import { useForm } from "react-hook-form";
import type { z } from "zod";
import { OnboardingLinks } from "../onboarding-links";

const formSchema = createAppRequestSchema.omit({ projectId: true });
type FormValues = z.infer<typeof formSchema>;

type CreateAppStepProps = {
  projectId: string;
  onAppCreated: (id: string) => void;
};

export const CreateAppStep = ({ projectId, onAppCreated }: CreateAppStepProps) => {
  const { next } = useStepWizard();
  const { gated, openPaywall, planGate } = useDeployActionGate();

  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();

  const {
    register,
    handleSubmit,
    setValue,
    setError,
    formState: { errors, isSubmitting, isValid },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: { name: "", slug: "" },
    mode: "onChange",
  });

  const onSubmitForm = async (values: FormValues) => {
    // Without a Compute plan, block app creation and surface the paywall.
    if (gated) {
      openPaywall();
      return;
    }
    try {
      const tx = collection.apps.insert({
        projectId,
        name: values.name,
        slug: values.slug,
        defaultBranch: "main",
        repositoryFullName: null,
        currentDeploymentId: null,
        isRolledBack: false,
        id: SERVER_PLACEHOLDER,
        latestDeploymentId: null,
        author: SERVER_PLACEHOLDER,
        authorAvatar: SERVER_PLACEHOLDER,
        branch: SERVER_PLACEHOLDER,
        commitTimestamp: Date.now(),
        commitTitle: SERVER_PLACEHOLDER,
        commitSha: null,
        forkRepositoryFullName: null,
        prNumber: null,
        domain: SERVER_PLACEHOLDER,
      });
      await tx.isPersisted.promise;
      const appId = (tx.metadata as { appId: string }).appId;

      try {
        const envs = await trpcClient.deploy.environment.list.query({ projectId });
        const appEnvs = envs.filter((e) => e.appId === appId);
        const regionNames = (availableRegions ?? []).map((r) => r.name);
        await Promise.all(
          appEnvs.map((env) => applyDefaultSettings(projectId, appId, env.id, regionNames)),
        );
      } catch (err) {
        toast.error("Failed to initialize settings", {
          description: err instanceof Error ? err.message : "An unexpected error occurred",
        });
      }

      onAppCreated(appId);
      next();
    } catch (error) {
      if (error instanceof DuplicateKeyError) {
        setError("slug", {
          type: "custom",
          message: "App with this slug already exists",
        });
      } else {
        console.error("Form submission error:", error);
      }
    }
  };

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setValue("slug", slugify(e.target.value));
  };

  return (
    <div className="w-full justify-center items-center flex flex-col">
      <div className="flex flex-col items-center border border-grayA-5 rounded-lg justify-center gap-4 py-[18px] px-4 min-w-[600px]">
        <form onSubmit={handleSubmit(onSubmitForm)} className="flex flex-col gap-4 w-full">
          <FormInput
            requirement="required"
            label="App Name"
            className="[&_input:first-of-type]:h-[36px]"
            autoFocus
            description="A descriptive name for your app."
            data-1p-ignore
            error={errors.name?.message}
            {...register("name", { onChange: handleNameChange })}
            placeholder="My Awesome App"
          />

          <FormInput
            requirement="required"
            label="Slug"
            className="[&_input:first-of-type]:h-[36px]"
            description="URL-friendly identifier for your app (auto-generated from name)."
            data-1p-ignore
            error={errors.slug?.message}
            {...register("slug")}
            placeholder="my-awesome-app"
          />

          <Button
            type="submit"
            variant="primary"
            size="xlg"
            disabled={isSubmitting || !isValid}
            loading={isSubmitting}
            className="w-full rounded-lg mt-2"
          >
            Create App
          </Button>
        </form>
      </div>
      <div className="mb-7" />
      <OnboardingLinks />
      {planGate}
    </div>
  );
};
