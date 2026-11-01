"use client";

import { environmentName } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/environment-label";
import { RepoNameLabel } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/apps/_components/settings/build-settings/github-settings/shared";
import { type App, type Environment, useAppScope, useEnvironments } from "@/lib/deploy-data";
import { UnsupportedDeployRefError, parseDeployRef } from "@/lib/deploy-ref";
import { githubUrl } from "@/lib/github-url";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Github, IconArrowUpRightOutline12 } from "@unkey/icons";
import {
  Item,
  ItemActions,
  ItemContent,
  ItemDescription,
  ItemMedia,
  ItemTitle,
  RelativeTime,
  Skeleton,
  toast,
} from "@unkey/ui";
import Link from "next/link";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { DeployRefPicker } from "./deploy-ref-picker";
import { DeploymentDialog } from "./deployment-dialog";
import { useCreateDeployment } from "./use-create-deployment";

const COMMIT_URL = /^https?:\/\/github\.com\/[^/]+\/([^/]+)\/commit\//;

function gitFormSchema(repoName: string | undefined) {
  return z.object({
    environmentId: z.string().min(1, "Environment is required"),
    ref: z
      .string()
      .trim()
      .min(1, "A commit or branch reference is required")
      .refine(
        (value) => {
          if (!value.startsWith("http")) {
            return true;
          }
          const urlMatch = value.match(COMMIT_URL);
          return Boolean(urlMatch) && (!repoName || urlMatch?.[1] === repoName);
        },
        {
          message: repoName
            ? `Paste a commit URL from ${repoName} or a fork of it`
            : "Paste a GitHub commit URL",
        },
      ),
  });
}

type GitFormValues = z.infer<ReturnType<typeof gitFormSchema>>;

function environmentIdForRef(
  ref: string,
  productionBranch: string,
  environments: readonly Pick<Environment, "id" | "kind">[],
): string {
  const kind = ref.trim() === productionBranch ? "production" : "preview";
  return environments.find((env) => env.kind === kind)?.id ?? "";
}

type GitDeploymentFormProps = {
  app: App;
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
};

export function GitDeploymentForm({ app, isOpen, onOpenChange }: GitDeploymentFormProps) {
  const { projectId, appId } = useAppScope();
  const { data: environments } = useEnvironments();
  const repositoryFullName = app.repositoryFullName ?? null;
  const [owner, repo] = repositoryFullName?.split("/") ?? [];
  const productionBranch = app.defaultBranch ?? "main";

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
      defaultBranch: productionBranch,
    },
    {
      enabled: isOpen && Boolean(installationId) && Boolean(owner) && Boolean(repo),
      onError: () => {
        toast.error("Failed to load repository details");
      },
    },
  );
  const repoDetailsLoading = installations.isInitialLoading || repoDetails.isInitialLoading;

  const {
    handleSubmit,
    setValue,
    reset,
    watch,
    formState: { errors, isValid, isSubmitting },
  } = useForm<GitFormValues>({
    resolver: zodResolver(gitFormSchema(repo)),
    mode: "onChange",
    defaultValues: { ref: "", environmentId: "" },
  });

  const close = () => {
    onOpenChange(false);
    reset();
  };
  const createDeployment = useCreateDeployment(close);

  const ref = watch("ref");
  const environmentId = watch("environmentId");
  const target = environments.find((env) => env.id === environmentId);

  function onSubmit(values: GitFormValues) {
    const environment = environments.find((env) => env.id === values.environmentId);
    if (!environment) {
      return;
    }
    try {
      createDeployment.mutate({ environment: environment.slug, git: parseDeployRef(values.ref) });
    } catch (error) {
      if (error instanceof UnsupportedDeployRefError) {
        toast.error(error.message);
        return;
      }
      throw error;
    }
  }

  return (
    <DeploymentDialog
      isOpen={isOpen}
      onOpenChange={(open) => (open ? onOpenChange(true) : close())}
      subTitle="Deploy from a specific commit or branch reference"
      submitLabel={target ? `Deploy to ${environmentName(target.slug)}` : "Deploy"}
      canSubmit={isValid}
      isSubmitting={createDeployment.isLoading || isSubmitting}
      onSubmit={handleSubmit(onSubmit)}
    >
      {repositoryFullName && (
        <RepoSourceRow
          fullName={repositoryFullName}
          pushedAt={repoDetails.data?.pushedAt ?? null}
          loading={repoDetailsLoading}
        />
      )}
      <DeployRefPicker
        value={ref}
        environmentId={environmentId}
        productionBranch={productionBranch}
        source={installationId && owner && repo ? { projectId, installationId, owner, repo } : null}
        branches={repoDetails.data?.branches ?? []}
        branchesLoading={repoDetailsLoading}
        refError={errors.ref?.message}
        environmentError={errors.environmentId?.message}
        onRefChange={(next) => {
          setValue("ref", next, { shouldValidate: true, shouldDirty: true });
          setValue("environmentId", environmentIdForRef(next, productionBranch, environments), {
            shouldValidate: true,
          });
        }}
        onEnvironmentChange={(id) => setValue("environmentId", id, { shouldValidate: true })}
      />
    </DeploymentDialog>
  );
}

function RepoSourceRow({
  fullName,
  pushedAt,
  loading,
}: {
  fullName: string;
  pushedAt: string | null;
  loading: boolean;
}) {
  const repoUrl = githubUrl.repo(fullName);

  return (
    <Item
      variant="outline"
      render={
        repoUrl ? (
          <Link href={new URL(repoUrl)} target="_blank" rel="noopener noreferrer" />
        ) : undefined
      }
    >
      <ItemMedia>
        <Github />
      </ItemMedia>
      <ItemContent>
        <ItemTitle>
          <RepoNameLabel fullName={fullName} />
        </ItemTitle>
        {loading ? (
          <div className="flex h-4 items-center">
            <Skeleton className="h-3 w-24 rounded" />
          </div>
        ) : (
          <ItemDescription>
            {pushedAt ? (
              <>
                Last pushed <RelativeTime value={pushedAt} />
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
  );
}
