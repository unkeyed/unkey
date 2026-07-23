"use client";
import { collection } from "@/lib/collections";
import { trpcClient } from "@/lib/collections/client";
import { createAppRequestSchema } from "@/lib/collections/deploy/apps";
import { buildDefaultSettingsMutations } from "@/lib/collections/deploy/environment-settings";
import { SERVER_PLACEHOLDER } from "@/lib/collections/deploy/utils";
import { slugify } from "@/lib/slugify";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { DuplicateKeyError } from "@tanstack/react-db";
import { Github } from "@unkey/icons";
import { Button, FormInput, toast, useStepWizard } from "@unkey/ui";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { OnboardingLinks } from "../onboarding-links";

// Only explicit GitHub references count as GitHub. A bare `owner/repo` is
// indistinguishable from a Docker Hub image, so everything else is an image.
const GITHUB_HTTP_RE = /^(?:https?:\/\/)?(?:www\.)?github\.com\/[^/\s]+\/[^/\s?#]+/i;
const GITHUB_SSH_RE = /^git@github\.com:[^/\s]+\/[^\s]+$/i;

const isGithubSource = (source: string) =>
  GITHUB_HTTP_RE.test(source) || GITHUB_SSH_RE.test(source);

const formSchema = createAppRequestSchema.omit({ projectId: true, dockerImage: true }).extend({
  source: z
    .string()
    .trim()
    .max(512, "Source is too long")
    .regex(/^\S*$/, "Source cannot contain whitespace")
    .optional(),
});
type FormValues = z.infer<typeof formSchema>;

type CreateAppStepProps = {
  projectId: string;
  onAppCreated: (id: string) => void;
  onSourceTypeChange: (sourceType: "docker_image" | "github") => void;
};

export const CreateAppStep = ({
  projectId,
  onAppCreated,
  onSourceTypeChange,
}: CreateAppStepProps) => {
  const { goTo } = useStepWizard();

  const { data: availableRegions } = trpc.deploy.environmentSettings.getAvailableRegions.useQuery();

  const {
    register,
    handleSubmit,
    setValue,
    setError,
    formState: { errors, isSubmitting },
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: { name: "", slug: "", source: "" },
    mode: "onChange",
  });

  const createApp = async (values: FormValues, dockerImage: string | null) => {
    const tx = collection.apps.insert({
      projectId,
      name: values.name,
      slug: values.slug,
      sourceType: "docker_image",
      defaultDockerImage: dockerImage,
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
      const mutations = appEnvs.flatMap((env) =>
        buildDefaultSettingsMutations(env.id, availableRegions ?? []),
      );
      if (mutations.length > 0) {
        await Promise.all(mutations);
      }
    } catch (err) {
      toast.error("Failed to initialize settings", {
        description: err instanceof Error ? err.message : "An unexpected error occurred",
      });
    }

    return appId;
  };

  const handleCreateError = (error: unknown) => {
    if (error instanceof DuplicateKeyError) {
      setError("slug", {
        type: "custom",
        message: "App with this slug already exists",
      });
    } else {
      console.error("Form submission error:", error);
    }
  };

  const onSubmitForm = async (values: FormValues) => {
    const source = values.source?.trim() ?? "";
    if (!source) {
      setError("source", {
        type: "custom",
        message: "Enter a Docker image or GitHub repository URL",
      });
      return;
    }

    try {
      if (isGithubSource(source)) {
        const appId = await createApp(values, null);
        onAppCreated(appId);
        onSourceTypeChange("github");
        goTo("select-repo");
        return;
      }

      const appId = await createApp(values, source);
      onAppCreated(appId);
      onSourceTypeChange("docker_image");
      goTo("configure-deployment");
    } catch (error) {
      handleCreateError(error);
    }
  };

  // Browse skips the source field entirely: create the app, then pick the
  // repo in the next step.
  const onBrowseGithub = async (values: FormValues) => {
    try {
      const appId = await createApp(values, null);
      onAppCreated(appId);
      onSourceTypeChange("github");
      goTo("select-repo");
    } catch (error) {
      handleCreateError(error);
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

          <div className="flex flex-col gap-2">
            <FormInput
              requirement="required"
              label="Source"
              className="[&_input:first-of-type]:h-[36px]"
              description="A Docker image to deploy, or a GitHub repository URL."
              data-1p-ignore
              error={errors.source?.message}
              {...register("source")}
              placeholder="ghcr.io/acme/my-app:latest"
            />
            <button
              type="button"
              className="self-start text-xs text-gray-10 hover:text-gray-12 transition-colors flex items-center gap-1.5"
              disabled={isSubmitting}
              onClick={handleSubmit(onBrowseGithub)}
            >
              <Github className="size-3" iconSize="sm-thin" />
              or browse your GitHub repositories
            </button>
          </div>

          <Button
            type="submit"
            variant="primary"
            size="xlg"
            disabled={isSubmitting}
            loading={isSubmitting}
            className="w-full rounded-lg mt-2"
          >
            Create App
          </Button>
        </form>
      </div>
      <div className="mb-7" />
      <OnboardingLinks />
    </div>
  );
};
