"use client";

import { queryClient } from "@/lib/collections/client";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronDown, CodeBranch, Plus } from "@unkey/icons";
import {
  Button,
  FormDescription,
  FormInput,
  FormLabel,
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
import { RepoDisplay } from "../../../_components/list/repo-display";
import { useProjectData } from "../data-provider";

const DynamicDialogContainer = dynamic(
  () =>
    import("@unkey/ui").then((mod) => ({
      default: mod.DialogContainer,
    })),
  { ssr: false },
);

const formSchema = z.object({
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
        return /^https?:\/\/github\.com\/[^/]+\/[^/]+\/(tree|commit)\//.test(val);
      },
      { message: "Enter a branch name, commit SHA, or GitHub URL (tree/commit)" },
    ),
});

type Props = {
  defaultOpen?: boolean;
};

export const CreateDeploymentButton = ({
  defaultOpen,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & Props) => {
  const router = useRouter();
  const params = useParams<{ workspaceSlug: string }>();
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);
  const { projectId, project, environments } = useProjectData();

  const repositoryFullName = project?.repositoryFullName ?? null;
  const [owner, repo] = repositoryFullName?.split("/") ?? [];
  const defaultBranch = project?.branch ?? "main";

  const installations = trpc.github.getInstallations.useQuery(
    { projectId },
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
    environments.find((e) => e.slug === "production")?.slug ?? environments[0]?.slug ?? "";

  const {
    register,
    handleSubmit,
    setValue,
    reset,
    control,
    formState: { errors, isValid, isSubmitting },
  } = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
    defaultValues: {
      environment: defaultEnvironmentSlug,
    },
  });

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
        `/${params.workspaceSlug}/projects/${projectId}/deployments/${data.deploymentId}`,
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
      environmentSlug: values.environment,
      gitRef: values.name,
    });
  }

  return (
    <>
      <Button {...rest} variant="outline" className="size-7" onClick={() => setIsOpen(true)}>
        <Plus iconSize="sm-regular" />
      </Button>
      <DynamicDialogContainer
        isOpen={isOpen}
        onOpenChange={setIsOpen}
        title="Create Deployment"
        subTitle="Deploy from a specific commit or branch reference"
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
            <FormInput
              label="Commit or Branch Reference"
              className="min-h-9"
              description={
                repositoryFullName
                  ? `Paste a valid commit reference to create a new deployment in addition to those auto-generated from ${repositoryFullName}.`
                  : "Paste a valid commit or branch reference to create a new deployment."
              }
              error={errors.name?.message}
              {...register("name")}
              placeholder={
                repositoryFullName
                  ? `https://github.com/${repositoryFullName}`
                  : "Enter a commit SHA or branch name"
              }
            />
          </form>

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
