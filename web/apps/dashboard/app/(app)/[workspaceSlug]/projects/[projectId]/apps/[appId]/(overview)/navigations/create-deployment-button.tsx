"use client";

import { environmentName } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { RepoNameLabel } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/build-settings/github-settings/shared";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { collection } from "@/lib/collections";
import { UnsupportedDeployRefError, parseDeployRef } from "@/lib/deploy-ref";
import { sanitizeImageRef, validateImageRef } from "@/lib/docker-image-ref";
import { githubUrl } from "@/lib/github-url";
import { trpc } from "@/lib/trpc/client";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { useMutation } from "@tanstack/react-query";
import {
  Github,
  IconArrowUpRightOutline12,
  IconChevronDownOutline12,
  IconPlusOutline12,
} from "@unkey/icons";
import { match } from "@unkey/match";
import {
  Button,
  FormDescription,
  FormInput,
  FormLabel,
  InfoTooltip,
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
  RelativeTime,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  Skeleton,
  TimestampInfo,
  toast,
} from "@unkey/ui";
import dynamic from "next/dynamic";
import type React from "react";
import { useEffect, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { useAppId, useProjectData } from "../data-provider";
import { useRecentDeployments } from "../hooks/use-recent-deployments";
import { environmentForRef } from "./deploy-ref-model";
import { DeployRefPicker } from "./deploy-ref-picker";

// Soft cap on the past-image list; older images are still deployable by
// pasting their reference into the input.
const MAX_IMAGE_ROWS = 10;

// Statuses proving the image deployed successfully at some point; in-flight
// and failed deployments are excluded.
const DEPLOYED_STATUSES = new Set(["ready", "stopped", "superseded"]);

const DynamicDialogContainer = dynamic(
  () =>
    import("@unkey/ui").then((mod) => ({
      default: mod.DialogContainer,
    })),
  { ssr: false },
);

function createFormSchema(repoName?: string, imageMode = false) {
  if (imageMode) {
    return z.object({
      environment: z.string().min(1, "Environment is required"),
      name: z
        .string()
        .trim()
        .superRefine((value, ctx) => {
          const result = validateImageRef(sanitizeImageRef(value));
          if (!result.ok) {
            ctx.addIssue({ code: "custom", message: result.error });
          }
        }),
    });
  }
  return z.object({
    environment: z.string().min(1, "Environment is required"),
    name: z
      .string()
      .trim()
      .min(1, "A commit or branch reference is required")
      .refine(
        (val) => {
          if (!val.startsWith("http")) {
            return true;
          }
          const urlMatch = val.match(/^https?:\/\/github\.com\/[^/]+\/([^/]+)\/commit\//);
          if (!urlMatch) {
            return false;
          }
          if (repoName && urlMatch[1] !== repoName) {
            return false;
          }
          return true;
        },
        {
          message: repoName
            ? `Paste a commit URL from ${repoName} or a fork of it`
            : "Paste a GitHub commit URL",
        },
      ),
  });
}

type Props = {
  defaultOpen?: boolean;
  renderTrigger?: (props: { onClick: () => void }) => React.ReactNode;
};

export const CreateDeploymentButton = ({
  defaultOpen,
  renderTrigger,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & Props) => {
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);
  const { projectId, environments, refetchDeployments } = useProjectData();
  const appId = useAppId();
  const { gated, openPaywall, planGate } = useDeployActionGate();

  // Without a Compute plan the trigger opens the paywall instead of the create
  // dialog; the backend still rejects the mutation either way.
  const openDeploymentDialog = () => (gated ? openPaywall() : setIsOpen(true));

  // Repo connections are per-app, not per-project; the project-level
  // repositoryFullName is just some app's connection in this project.
  const appQuery = useLiveQuery(
    (q) =>
      q
        .from({ app: collection.apps })
        .where(({ app }) => and(eq(app.projectId, projectId), eq(app.id, appId))),
    [projectId, appId],
  );
  const app = appQuery.data?.[0];

  const repositoryFullName = app?.repositoryFullName ?? null;
  const repoUrl = githubUrl.repo(repositoryFullName);
  const [owner, repo] = repositoryFullName?.split("/") ?? [];
  const defaultBranch = app?.defaultBranch ?? "main";
  const deploymentSource = app
    ? match(app.sourceType)
        .returnType<"git" | "oci">()
        .with("git", () => "git")
        .with("oci", () => "oci")
        .with("unknown", () => (repositoryFullName ? "git" : "oci"))
        .exhaustive()
    : null;
  const sourceFlags = deploymentSource
    ? match(deploymentSource)
        .with("git", () => ({ isGitApp: true, isImageApp: false }))
        .with("oci", () => ({ isGitApp: false, isImageApp: true }))
        .exhaustive()
    : { isGitApp: false, isImageApp: false };
  const { deployments } = useRecentDeployments({ enabled: isOpen && sourceFlags.isImageApp });
  const { isGitApp, isImageApp } = sourceFlags;

  const installations = trpc.github.getInstallations.useQuery(
    { projectId, appId },
    { enabled: isOpen && isGitApp && Boolean(repositoryFullName) },
  );

  const installationId = installations.data?.repoConnection?.installationId;

  const repoDetails = trpc.github.getRepositoryDetails.useQuery(
    {
      projectId,
      installationId: installationId ?? 0,
      owner: owner ?? "",
      repo: repo ?? "",
      defaultBranch,
    },
    {
      enabled: isOpen && Boolean(installationId) && Boolean(owner) && Boolean(repo),
      onError: () => {
        toast.error("Failed to load repository details");
      },
    },
  );

  const branches = repoDetails.data?.branches ?? [];

  const defaultEnvironmentSlug = isImageApp
    ? (environments.find((e) => e.kind === "preview")?.slug ?? environments[0]?.slug ?? "")
    : "";
  const repoDetailsLoading = installations.isInitialLoading || repoDetails.isInitialLoading;

  const formSchema = createFormSchema(repo, isImageApp);

  const {
    register,
    handleSubmit,
    setValue,
    reset,
    watch,
    getFieldState,
    control,
    formState: { errors, isValid, isSubmitting },
  } = useForm<z.infer<ReturnType<typeof createFormSchema>>>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
    defaultValues: {
      environment: defaultEnvironmentSlug,
    },
  });

  const nameValue = watch("name") ?? "";
  const imageRef = sanitizeImageRef(nameValue);
  const imageValidation = validateImageRef(imageRef);
  const environmentValue = watch("environment") ?? "";
  const targetEnvironment = environments.find((env) => env.slug === environmentValue);
  const deployLabel =
    isGitApp && targetEnvironment
      ? `Deploy to ${environmentName(targetEnvironment.slug)}`
      : "Deploy";

  useEffect(() => {
    if (defaultEnvironmentSlug) {
      setValue("environment", defaultEnvironmentSlug, { shouldValidate: true });
    }
  }, [defaultEnvironmentSlug, setValue]);

  useEffect(() => {
    if (isOpen && isImageApp && app?.imageReference && !getFieldState("name").isDirty) {
      setValue("name", app.imageReference, { shouldValidate: true });
    }
  }, [app?.imageReference, getFieldState, isImageApp, isOpen, setValue]);

  const createDeployment = useMutation({
    mutationFn: async (source: {
      environment: string;
      git?: ReturnType<typeof parseDeployRef>;
      oci?: string;
    }) => {
      const response = await getUnkeyClient().deployments.createDeploymentV3({
        project: projectId,
        app: appId,
        environment: source.environment,
        ...(source.oci ? { oci: { image: source.oci } } : { git: source.git ?? {} }),
      });
      return { deploymentId: response.data.deploymentId };
    },
    async onSuccess() {
      toast.success("Deployment has been created");
      reset({ environment: defaultEnvironmentSlug });
      setIsOpen(false);
      refetchDeployments();
      await collection.apps.utils.refetch();
    },
    onError(err) {
      console.error(err);
      toast.error(getErrorMessage(err));
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (!deploymentSource) {
      return;
    }

    if (deploymentSource === "oci") {
      createDeployment.mutate({
        environment: values.environment,
        oci: imageRef,
      });
      return;
    }

    try {
      createDeployment.mutate({
        environment: values.environment,
        git: parseDeployRef(values.name),
      });
    } catch (error) {
      if (error instanceof UnsupportedDeployRefError) {
        toast.error(error.message);
        return;
      }
      throw error;
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
    <>
      {renderTrigger ? (
        renderTrigger({ onClick: openDeploymentDialog })
      ) : (
        <NavbarActionButton
          {...rest}
          color="default"
          variant="outline"
          className="size-7"
          onClick={openDeploymentDialog}
        >
          <IconPlusOutline12 />
        </NavbarActionButton>
      )}
      {planGate}
      <DynamicDialogContainer
        isOpen={isOpen}
        onOpenChange={(open) => {
          setIsOpen(open);
          if (!open) {
            reset({ environment: defaultEnvironmentSlug });
          }
        }}
        title="Create Deployment"
        subTitle={
          isImageApp
            ? "Deploy a prebuilt image or redeploy a previous one"
            : "Deploy from a specific commit or branch reference"
        }
        preventAutoFocus
        footer={
          <div className="w-full flex flex-col gap-2 items-center justify-center">
            <Button
              type="submit"
              form="create-deployment-form"
              variant="primary"
              size="xlg"
              disabled={
                createDeployment.isLoading || isSubmitting || !isValid || environments.length === 0
              }
              loading={createDeployment.isLoading || isSubmitting}
              className="w-full rounded-lg"
            >
              {deployLabel}
            </Button>
          </div>
        }
      >
        <div className="flex flex-col gap-5 py-2">
          {isGitApp && repositoryFullName && (
            <Item
              variant="outline"
              render={
                repoUrl ? (
                  // biome-ignore lint/a11y/useAnchorContent: Item renders its children into this anchor
                  <a href={repoUrl} target="_blank" rel="noopener noreferrer" />
                ) : undefined
              }
            >
              <ItemMedia>
                <Github />
              </ItemMedia>
              <ItemContent>
                <ItemTitle>
                  <RepoNameLabel fullName={repositoryFullName} />
                </ItemTitle>
                {repoDetailsLoading ? (
                  <div className="flex h-4 items-center">
                    <Skeleton className="h-3 w-24 rounded" />
                  </div>
                ) : (
                  <ItemDescription>
                    {repoDetails.data?.pushedAt ? (
                      <>
                        Last pushed <RelativeTime value={repoDetails.data.pushedAt} />
                      </>
                    ) : (
                      "Connected repository"
                    )}
                  </ItemDescription>
                )}
              </ItemContent>
              {repoUrl && (
                <ItemActions>
                  <IconArrowUpRightOutline12 />
                </ItemActions>
              )}
            </Item>
          )}

          <form
            id="create-deployment-form"
            onSubmit={handleSubmit(onSubmit)}
            className="flex flex-col gap-5"
          >
            {deploymentSource &&
              match(deploymentSource)
                .with("git", () => (
                  <DeployRefPicker
                    value={nameValue}
                    environment={environmentValue}
                    environments={environments}
                    productionBranch={defaultBranch}
                    source={
                      installationId && owner && repo
                        ? { projectId, installationId, owner, repo }
                        : null
                    }
                    branches={branches}
                    branchesLoading={repoDetailsLoading}
                    refError={errors.name?.message}
                    environmentError={errors.environment?.message}
                    onRefChange={(ref) => {
                      setValue("name", ref, { shouldValidate: true, shouldDirty: true });
                      setValue(
                        "environment",
                        environmentForRef(ref, defaultBranch, environments) ?? "",
                        {
                          shouldValidate: true,
                        },
                      );
                    }}
                    onEnvironmentChange={(slug) =>
                      setValue("environment", slug, { shouldValidate: true })
                    }
                  />
                ))
                .with("oci", () => (
                  <>
                    <fieldset className="flex flex-col gap-2 border-0 m-0 p-0">
                      <FormLabel label="Environment" htmlFor="environment-select" />
                      <Controller
                        control={control}
                        name="environment"
                        render={({ field }) => (
                          <Select value={field.value} onValueChange={field.onChange}>
                            <SelectTrigger
                              id="environment-select"
                              className="capitalize"
                              variant={errors.environment ? "error" : "default"}
                              rightIcon={
                                <IconChevronDownOutline12 className="absolute right-3 size-3 opacity-70" />
                              }
                            >
                              <SelectValue placeholder="Select environment" />
                            </SelectTrigger>
                            <SelectContent>
                              {environments.map((env) => (
                                <SelectItem key={env.id} value={env.slug} className="capitalize">
                                  {env.slug}
                                </SelectItem>
                              ))}
                            </SelectContent>
                          </Select>
                        )}
                      />
                      <FormDescription
                        description="Target environment for this deployment."
                        descriptionId="environment-select-description"
                        errorId="environment-select-error"
                        error={errors.environment?.message}
                      />
                    </fieldset>
                    <div className="flex flex-col gap-2">
                      <FormInput
                        label="Image Reference"
                        className="min-h-9"
                        autoComplete="off"
                        data-1p-ignore
                        description="Paste a container image reference to deploy, or pick a previously deployed image below."
                        error={errors.name?.message}
                        {...register("name", {
                          onBlur: () => setValue("name", imageRef, { shouldValidate: true }),
                        })}
                        onPaste={(e) => {
                          const cleaned = sanitizeImageRef(e.clipboardData.getData("text"));
                          e.preventDefault();
                          setValue("name", cleaned, { shouldValidate: true });
                        }}
                        placeholder="registry.example.com/my-app:v1.2.3"
                      />
                      {imageValidation.ok && imageValidation.warning ? (
                        <output className="text-warning-11 text-sm">
                          {imageValidation.warning}
                        </output>
                      ) : null}
                    </div>
                  </>
                ))
                .exhaustive()}
          </form>

          {isImageApp && imageRows.length > 0 && (
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
                        setValue("name", deployment.resolvedImage ?? "", {
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
        </div>
      </DynamicDialogContainer>
    </>
  );
};
