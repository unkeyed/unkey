"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { collection } from "@/lib/collections";
import {
  type CreateProjectRequestSchema,
  createProjectRequestSchema,
} from "@/lib/collections/deploy/projects";
import { SERVER_PLACEHOLDER } from "@/lib/collections/deploy/utils";
import { routes } from "@/lib/navigation/routes";
import { slugify } from "@/lib/slugify";
import { zodResolver } from "@hookform/resolvers/zod";
import { DuplicateKeyError } from "@tanstack/react-db";
import { Button, Card, FormInput, FullScreenContent } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type React from "react";
import { useTransition } from "react";
import { useForm } from "react-hook-form";

/**
 * Full-page create-project flow: a plain card centered with a max-width and
 * px-4 gutter so it stays usable on small screens. The (app) layout hides the
 * sidebar for this route the same way it does for apps/new.
 */
export default function NewProjectPage() {
  const router = useRouter();
  const workspace = useWorkspaceNavigation();
  const [isNavigating, startNavigation] = useTransition();

  const {
    register,
    handleSubmit,
    setValue,
    setError,
    formState: { errors, isValid, isSubmitting },
  } = useForm<CreateProjectRequestSchema>({
    resolver: zodResolver(createProjectRequestSchema),
    defaultValues: { name: "", slug: "" },
    mode: "onChange",
  });

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setValue("slug", slugify(e.target.value));
  };

  async function onSubmit(values: CreateProjectRequestSchema) {
    try {
      const tx = collection.projects.insert({
        name: values.name,
        slug: values.slug,
        appCount: 0,
        apps: [],
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
        domain: SERVER_PLACEHOLDER,
        commitSha: null,
        forkRepositoryFullName: null,
        prNumber: null,
      });
      await tx.isPersisted.promise;
      const { projectId } = tx.metadata as { projectId: string };
      // Keep the submit button in its loading state until the app-create flow
      // has rendered; releasing it immediately leaves the user staring at a
      // finished form.
      startNavigation(() => {
        router.push(routes.projects.apps.new({ workspaceSlug: workspace.slug, projectId }));
      });
    } catch (error) {
      if (error instanceof DuplicateKeyError) {
        setError("slug", {
          type: "custom",
          message: "Project with this slug already exists",
        });
      } else {
        console.error("Form submission error:", error);
      }
    }
  }

  const isBusy = isSubmitting || isNavigating;

  return (
    <FullScreenContent className="px-4 py-8">
      <form onSubmit={handleSubmit(onSubmit)} className="w-full flex flex-col items-center">
        <Card className="max-w-[440px] shadow-xs p-6 flex flex-col gap-4">
          <div className="flex flex-col gap-1">
            <h2 className="text-lg font-semibold text-gray-12">Create a project</h2>
            <p className="text-sm text-gray-11">
              Projects group your apps and everything they need to run.
            </p>
          </div>
          <FormInput
            requirement="required"
            label="Project Name"
            className="[&_input:first-of-type]:h-[36px]"
            description="A descriptive name for your project."
            data-1p-ignore
            error={errors.name?.message}
            {...register("name", { onChange: handleNameChange })}
            placeholder="My Awesome Project"
          />
          <FormInput
            requirement="required"
            label="Slug"
            className="[&_input:first-of-type]:h-[36px]"
            description="URL-friendly identifier for your project (auto-generated from name)."
            data-1p-ignore
            error={errors.slug?.message}
            {...register("slug")}
            placeholder="my-awesome-project"
          />
          <div className="flex flex-col gap-2 mt-2">
            <Button
              type="submit"
              variant="primary"
              size="xlg"
              disabled={isBusy || !isValid}
              loading={isBusy}
              className="w-full rounded-lg"
            >
              Create Project
            </Button>
            <div className="text-gray-9 text-xs text-center">
              You'll be redirected to your new project after creation
            </div>
          </div>
        </Card>
      </form>
    </FullScreenContent>
  );
}
