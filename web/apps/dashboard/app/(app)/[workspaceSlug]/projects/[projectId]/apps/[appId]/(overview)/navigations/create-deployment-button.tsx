"use client";

import { RepoDisplay } from "@/app/(app)/[workspaceSlug]/projects/_components/list/repo-display";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { collection } from "@/lib/collections";
import { queryClient } from "@/lib/collections/client";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { and, eq, useLiveQuery } from "@tanstack/react-db";
import { ChevronDown, CodeBranch, Plus } from "@unkey/icons";
import {
  Button,
  FormDescription,
  FormInput,
  FormLabel,
  InfoTooltip,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  TimestampInfo,
  toast,
} from "@unkey/ui";
import dynamic from "next/dynamic";
import { useParams, useRouter } from "next/navigation";
import type React from "react";
import { useEffect, useState } from "react";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { useAppId, useProjectData } from "../data-provider";
import { parseForkRef } from "./parse-fork-ref";

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
        .min(1, "An image reference is required")
        .regex(/^\S+$/, "Image reference cannot contain spaces"),
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
          const urlMatch = val.match(
            /^https?:\/\/github\.com\/[^/]+\/([^/]+)\/(tree|commit|pull)\//,
          );
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
            ? `URL must point to a repository named "${repoName}" (the connected repo or a fork of it)`
            : "Enter a branch name (e.g. main), commit SHA, fork reference (owner:branch), or a GitHub URL to a branch, commit, or pull request",
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
  const router = useRouter();
  const params = useParams<{ workspaceSlug: string }>();
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);
  const { projectId, environments, deployments } = useProjectData();
  const appId = useAppId();

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
  const [owner, repo] = repositoryFullName?.split("/") ?? [];
  const defaultBranch = app?.defaultBranch ?? "main";
  const isCliApp = !appQuery.isLoading && app != null && !repositoryFullName;

  const installations = trpc.github.getInstallations.useQuery(
    { projectId, appId },
    { enabled: isOpen && Boolean(repositoryFullName) },
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

  const defaultEnvironmentSlug =
    environments.find((e) => e.slug === "preview")?.slug ?? environments[0]?.slug ?? "";

  const formSchema = createFormSchema(repo, isCliApp);

  const {
    register,
    handleSubmit,
    setValue,
    reset,
    watch,
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
  const detectedFork = parseForkRef(nameValue);
  const forkRepoName = detectedFork && repo ? `${detectedFork.forkOwner}/${repo}` : null;

  useEffect(() => {
    if (defaultEnvironmentSlug) {
      setValue("environment", defaultEnvironmentSlug, { shouldValidate: true });
    }
  }, [defaultEnvironmentSlug, setValue]);

  const createDeployment = trpc.deploy.deployment.create.useMutation({
    async onSuccess(data) {
      toast.success("Deployment has been created");
      reset();
      setIsOpen(false);
      await queryClient.invalidateQueries({ queryKey: ["deployments", projectId] });
      router.push(
        `/${params.workspaceSlug}/projects/${projectId}/apps/${appId}/deployments/${data.deploymentId}`,
      );
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    createDeployment.mutate({
      projectId,
      appId,
      environmentSlug: values.environment,
      ...(isCliApp
        ? { source: "image" as const, image: values.name }
        : { source: "git" as const, gitRef: values.name }),
    });
  }

  // Past successfully deployed prebuilt images, deduped by image ref
  // (deployments are ordered newest-first, so the latest deployment wins).
  const imageRows = deployments
    .filter(
      (d, i, all) =>
        d.image &&
        DEPLOYED_STATUSES.has(d.status) &&
        all.findIndex((o) => o.image === d.image) === i,
    )
    .slice(0, MAX_IMAGE_ROWS);

  return (
    <>
      {renderTrigger ? (
        renderTrigger({ onClick: () => setIsOpen(true) })
      ) : (
        <NavbarActionButton
          {...rest}
          color="default"
          variant="outline"
          className="size-7"
          onClick={() => setIsOpen(true)}
        >
          <Plus iconSize="sm-regular" />
        </NavbarActionButton>
      )}
      <DynamicDialogContainer
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        title="Create Deployment"
        subTitle={
          isCliApp
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
              Deploy
            </Button>
          </div>
        }
      >
        <div className="flex flex-col gap-10 py-3">
          {repositoryFullName && (
            <div className="flex items-start gap-2 flex-col">
              <RepoDisplay
                url={`https://github.com/${repositoryFullName}`}
                className="bg-grayA-4 px-1.5 font-medium text-xs text-gray-12 rounded-full min-h-[22px]"
              />
              {repoDetails.data?.pushedAt ? (
                <span className="text-xs text-gray-10">
                  Last pushed{" "}
                  <TimestampInfo
                    value={repoDetails.data.pushedAt}
                    displayType="relative"
                    className="font-medium underline decoration-dotted text-gray-12"
                  />
                </span>
              ) : repoDetails.isLoading ? (
                <div className="h-4 w-16 bg-grayA-3 rounded animate-pulse" />
              ) : null}
            </div>
          )}

          <form
            id="create-deployment-form"
            onSubmit={handleSubmit(onSubmit)}
            className="flex flex-col gap-6"
          >
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
                      rightIcon={<ChevronDown className="absolute right-3 size-3 opacity-70" />}
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
                label={isCliApp ? "Image Reference" : "Commit or Branch Reference"}
                className="min-h-9"
                description={
                  isCliApp
                    ? "Paste a Docker image reference to deploy, or pick a previously deployed image below."
                    : repositoryFullName
                      ? "Paste a commit, branch, PR URL, or fork reference (e.g. fork-owner:branch) to deploy."
                      : "Paste a valid commit, branch reference, or PR URL to create a new deployment."
                }
                error={errors.name?.message}
                {...register("name")}
                placeholder={
                  isCliApp
                    ? "registry.example.com/my-app:v1.2.3"
                    : repositoryFullName
                      ? `https://github.com/${repositoryFullName}/tree/${defaultBranch}`
                      : "Enter a commit SHA, branch, or PR URL"
                }
              />
              {forkRepoName && (
                <div className="flex items-center gap-1.5 bg-amber-3 border border-amber-6 rounded-md px-2.5 py-1.5 w-fit">
                  <CodeBranch iconSize="sm-regular" className="shrink-0 text-amber-11" />
                  <span className="text-xs text-amber-11">
                    Deploying from fork:{" "}
                    <span className="font-medium text-amber-12">{forkRepoName}</span>
                  </span>
                </div>
              )}
            </div>
          </form>

          {isCliApp && imageRows.length > 0 && (
            <div className="flex flex-col divide-y divide-gray-4 rounded-md border border-gray-4 overflow-hidden">
              {imageRows.map((deployment) => (
                <button
                  key={deployment.id}
                  type="button"
                  onClick={() => setValue("name", deployment.image ?? "", { shouldValidate: true })}
                  className="flex items-center justify-between px-3 py-2 bg-grayA-2 hover:bg-grayA-3 transition-colors cursor-pointer text-[13px] text-grayA-11"
                >
                  <span className="flex items-center gap-1.5 min-w-0 max-w-[300px]">
                    <InfoTooltip
                      content={deployment.image}
                      asChild
                      position={{ align: "start", side: "top" }}
                    >
                      <span className="truncate">{deployment.image}</span>
                    </InfoTooltip>
                  </span>
                  <TimestampInfo
                    value={deployment.createdAt}
                    displayType="relative"
                    className="text-gray-11 shrink-0 ml-3"
                  />
                </button>
              ))}
            </div>
          )}

          {repositoryFullName && (
            <div className="flex flex-col divide-y divide-gray-4 rounded-md border border-gray-4">
              {repoDetails.isLoading &&
                Array.from({ length: 5 }).map((_, i) => (
                  <div
                    key={`skeleton-${
                      // biome-ignore lint/suspicious/noArrayIndexKey: skeleton placeholders
                      i
                    }`}
                    className="flex items-center justify-between px-3 py-2 h-[36.5px]"
                  >
                    <span className="flex items-center gap-1.5">
                      <span className="h-4 w-4 bg-gray-3 rounded animate-pulse" />
                      <span className="h-4 w-24 bg-gray-3 rounded animate-pulse" />
                    </span>
                    <span className="h-3 w-12 bg-gray-3 rounded animate-pulse" />
                  </div>
                ))}
              {branches.map((branch) => (
                <button
                  key={branch.name}
                  type="button"
                  onClick={() => setValue("name", branch.name, { shouldValidate: true })}
                  className="flex items-center justify-between px-3 py-2 bg-grayA-2 hover:bg-grayA-3 transition-colors cursor-pointer text-[13px] text-grayA-11"
                >
                  <span className="flex items-center gap-1.5 min-w-0 max-w-[300px]">
                    <CodeBranch iconSize="sm-regular" className="shrink-0 text-gray-12" />
                    <span className="truncate">{branch.name}</span>
                  </span>
                  {branch.lastPushDate && (
                    <TimestampInfo
                      value={branch.lastPushDate}
                      displayType="relative"
                      className="text-gray-11 shrink-0 ml-3"
                    />
                  )}
                </button>
              ))}
            </div>
          )}
        </div>
      </DynamicDialogContainer>
    </>
  );
};
