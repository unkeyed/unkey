"use client";

import { trpcClient } from "@/lib/collections/client";
import { Button, FormInput, toast } from "@unkey/ui";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useMemo, useState } from "react";
import { type CloneParams, slugify } from "./parse-params";

type DeployFormProps = {
  workspaceSlug: string;
  installationId: number;
  repository: {
    id: number;
    fullName: string;
    defaultBranch: string;
    htmlUrl: string;
  };
  params: CloneParams;
};

export function DeployForm({ workspaceSlug, installationId, repository, params }: DeployFormProps) {
  const router = useRouter();
  const initialName = params.projectName ?? repository.fullName.split("/")[1] ?? "";

  const [name, setName] = useState(initialName);
  const [slug, setSlug] = useState(slugify(initialName));
  const [branch, setBranch] = useState(params.branch ?? repository.defaultBranch);
  const [rootDirectory, setRootDirectory] = useState(params.rootDirectory ?? "");
  const [dockerfile, setDockerfile] = useState(params.dockerfile ?? "");
  const [envValues, setEnvValues] = useState<Record<string, string>>(() =>
    Object.fromEntries(params.envKeys.map((k) => [k, ""])),
  );
  const [slugTouched, setSlugTouched] = useState(false);
  const [isSubmitting, setIsSubmitting] = useState(false);

  const errors = useMemo(
    () => validate({ name, slug, branch, envValues, envKeys: params.envKeys }),
    [name, slug, branch, envValues, params.envKeys],
  );
  const isValid = Object.keys(errors).length === 0;

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const next = e.target.value;
    setName(next);
    if (!slugTouched) {
      setSlug(slugify(next));
    }
  };

  const handleSlugChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setSlug(e.target.value);
    setSlugTouched(true);
  };

  const handleEnvChange = (key: string, value: string) => {
    setEnvValues((prev) => ({ ...prev, [key]: value }));
  };

  const onSubmit = async (e: React.FormEvent) => {
    e.preventDefault();
    if (!isValid || isSubmitting) {
      return;
    }
    setIsSubmitting(true);
    try {
      const { id: projectId } = await trpcClient.deploy.project.create.mutate({ name, slug });

      await trpcClient.github.selectRepository.mutate({
        projectId,
        repositoryId: repository.id,
        repositoryFullName: repository.fullName,
        installationId,
        selectedBranch: branch,
      });

      const { id: environmentId } = await trpcClient.deploy.project.getEnvironmentBySlug.query({
        projectId,
        slug: "production",
      });

      if (rootDirectory.trim().length > 0) {
        await trpcClient.deploy.environmentSettings.build.updateDockerContext.mutate({
          environmentId,
          dockerContext: rootDirectory.trim(),
        });
      }

      if (dockerfile.trim().length > 0) {
        await trpcClient.deploy.environmentSettings.build.updateDockerfile.mutate({
          environmentId,
          dockerfile: dockerfile.trim(),
        });
      }

      const variables = Object.entries(envValues)
        .map(([key, value]) => ({ key, value: value.trim() }))
        .filter((v) => v.value.length > 0)
        .map((v) => ({
          key: v.key,
          value: v.value,
          type: "writeonly" as const,
        }));

      if (variables.length > 0) {
        await trpcClient.deploy.envVar.create.mutate({ environmentId, variables });
      }

      const { deploymentId } = await trpcClient.deploy.deployment.create.mutate({
        projectId,
        environmentSlug: "production",
      });

      router.push(`/${workspaceSlug}/projects/${projectId}/deployments/${deploymentId}`);
    } catch (err) {
      const description = err instanceof Error ? err.message : "Unknown error";
      toast.error("Failed to deploy", { description });
      setIsSubmitting(false);
    }
  };

  return (
    <div className="min-h-screen flex items-center justify-center p-6">
      <div className="w-full max-w-2xl">
        <div className="mb-6 flex flex-col items-center text-center">
          <div className="text-2xl font-medium text-gray-12 leading-7">Unkey</div>
          <h1 className="mt-6 text-lg font-semibold text-gray-12">Deploy from GitHub</h1>
          <p className="mt-2 text-[13px] text-gray-10">
            You're deploying{" "}
            <a
              href={repository.htmlUrl}
              target="_blank"
              rel="noopener noreferrer"
              className="font-medium text-gray-12 underline underline-offset-2"
            >
              {repository.fullName}
            </a>{" "}
            to workspace <span className="font-medium text-gray-12">{workspaceSlug}</span>.
          </p>
        </div>

        <form
          onSubmit={onSubmit}
          className="flex flex-col gap-4 rounded-[14px] border border-grayA-5 p-5"
        >
          <FormInput
            requirement="required"
            label="Project name"
            description="A descriptive name for your project."
            value={name}
            onChange={handleNameChange}
            error={errors.name}
            placeholder="My awesome project"
            data-1p-ignore
          />

          <FormInput
            requirement="required"
            label="Slug"
            description="URL-friendly identifier (lowercase letters, numbers, hyphens)."
            value={slug}
            onChange={handleSlugChange}
            error={errors.slug}
            placeholder="my-awesome-project"
            data-1p-ignore
          />

          <FormInput
            requirement="required"
            label="Production branch"
            description="The branch to deploy when pushes hit the production environment."
            value={branch}
            onChange={(e) => setBranch(e.target.value)}
            error={errors.branch}
            placeholder={repository.defaultBranch}
          />

          <FormInput
            requirement="optional"
            label="Root directory"
            description="Build context for the Docker build. Use this for monorepos."
            value={rootDirectory}
            onChange={(e) => setRootDirectory(e.target.value)}
            placeholder="."
          />

          <FormInput
            requirement="optional"
            label="Dockerfile"
            description="Path to the Dockerfile, relative to the root directory."
            value={dockerfile}
            onChange={(e) => setDockerfile(e.target.value)}
            placeholder="Dockerfile"
          />

          {params.envKeys.length > 0 && (
            <div className="flex flex-col gap-2 pt-2">
              <div className="flex flex-col gap-1">
                <span className="text-[13px] font-medium text-gray-12">Environment variables</span>
                {params.envDescription && (
                  <span className="text-[12px] text-gray-10">{params.envDescription}</span>
                )}
                {params.envLink && (
                  <Link
                    href={params.envLink}
                    target="_blank"
                    rel="noopener noreferrer"
                    className="text-[12px] text-gray-11 underline underline-offset-2"
                  >
                    Where to find these values
                  </Link>
                )}
              </div>
              {params.envKeys.map((key) => (
                <FormInput
                  key={key}
                  requirement="required"
                  label={key}
                  value={envValues[key] ?? ""}
                  onChange={(e) => handleEnvChange(key, e.target.value)}
                  error={errors[`env.${key}`]}
                  type="password"
                />
              ))}
            </div>
          )}

          <Button
            type="submit"
            variant="primary"
            size="xlg"
            disabled={!isValid || isSubmitting}
            loading={isSubmitting}
            className="mt-2 w-full rounded-lg"
          >
            Deploy
          </Button>
        </form>
      </div>
    </div>
  );
}

function validate(args: {
  name: string;
  slug: string;
  branch: string;
  envValues: Record<string, string>;
  envKeys: string[];
}): Record<string, string> {
  const errors: Record<string, string> = {};
  if (!args.name.trim()) {
    errors.name = "Project name is required";
  }
  if (!args.slug.trim()) {
    errors.slug = "Slug is required";
  } else if (!/^[a-z0-9-]+$/.test(args.slug)) {
    errors.slug = "Only lowercase letters, numbers, and hyphens";
  }
  if (!args.branch.trim()) {
    errors.branch = "Branch is required";
  }
  for (const key of args.envKeys) {
    if (!args.envValues[key] || !args.envValues[key].trim()) {
      errors[`env.${key}`] = "Required";
    }
  }
  return errors;
}
