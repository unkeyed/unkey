"use client";

import { collection } from "@/lib/collections";
import {
  type CreateProjectRequestSchema,
  createProjectRequestSchema,
} from "@/lib/collections/deploy/projects";
import { SERVER_PLACEHOLDER } from "@/lib/collections/deploy/utils";
import { slugify } from "@/lib/slugify";
import { zodResolver } from "@hookform/resolvers/zod";
import { DuplicateKeyError } from "@tanstack/react-db";
import { Button, FormInput } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import type React from "react";
import { useForm } from "react-hook-form";

const DynamicDialogContainer = dynamic(
  () =>
    import("@unkey/ui").then((mod) => ({
      default: mod.DialogContainer,
    })),
  { ssr: false },
);

type Props = {
  isOpen: boolean;
  onOpenChange: (open: boolean) => void;
  workspaceSlug: string;
};

export const CreateProjectDialog = ({ isOpen, onOpenChange, workspaceSlug }: Props) => {
  const router = useRouter();
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
      router.push(`/${workspaceSlug}/projects/${projectId}/apps/new`);
      onOpenChange(false);
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

  return (
    <DynamicDialogContainer
      isOpen={isOpen}
      onOpenChange={onOpenChange}
      title="Create New Project"
      footer={
        <div className="w-full flex flex-col gap-2 items-center justify-center">
          <Button
            type="submit"
            form="create-project-form"
            variant="primary"
            size="xlg"
            disabled={isSubmitting || !isValid}
            loading={isSubmitting}
            className="w-full rounded-lg"
          >
            Create Project
          </Button>
          <div className="text-gray-9 text-xs">
            You'll be redirected to your new project after creation
          </div>
        </div>
      }
    >
      <form
        id="create-project-form"
        onSubmit={handleSubmit(onSubmit)}
        className="flex flex-col gap-4"
      >
        <FormInput
          requirement="required"
          label="Project Name"
          description="A descriptive name for your project."
          error={errors.name?.message}
          {...register("name", { onChange: handleNameChange })}
          placeholder="My Awesome Project"
        />
        <FormInput
          requirement="required"
          label="Slug"
          description="URL-friendly identifier for your project (auto-generated from name)."
          error={errors.slug?.message}
          {...register("slug")}
          placeholder="my-awesome-project"
        />
      </form>
    </DynamicDialogContainer>
  );
};
