"use client";

import { NavbarActionButton } from "@/components/navigation/action-button";
import { collection } from "@/lib/collections";
import {
  type CreateProjectRequestSchema,
  createProjectRequestSchema,
} from "@/lib/collections/deploy/projects";
import { SERVER_PLACEHOLDER } from "@/lib/collections/deploy/utils";
import { newAppPath } from "@/lib/navigation/routes";
import { slugify } from "@/lib/slugify";
import { zodResolver } from "@hookform/resolvers/zod";
import { DuplicateKeyError } from "@tanstack/react-db";
import { Plus } from "@unkey/icons";
import { Button, FormInput } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import type React from "react";
import { useState } from "react";
import { useForm } from "react-hook-form";

const DynamicDialogContainer = dynamic(
  () =>
    import("@unkey/ui").then((mod) => ({
      default: mod.DialogContainer,
    })),
  { ssr: false },
);

type Props = {
  defaultOpen?: boolean;
  workspaceSlug: string;
};

export const CreateProjectButton = ({
  defaultOpen,
  workspaceSlug,
  ...rest
}: React.ButtonHTMLAttributes<HTMLButtonElement> & Props) => {
  const [isOpen, setIsOpen] = useState(defaultOpen ?? false);
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
      });
      await tx.isPersisted.promise;
      const { projectId } = tx.metadata as { projectId: string };
      router.push(newAppPath({ workspaceSlug, projectId }));
      setIsOpen(false);
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
    <>
      <NavbarActionButton
        title="Create new project"
        {...rest}
        color="default"
        onClick={() => setIsOpen(true)}
      >
        <Plus />
        Create new project
      </NavbarActionButton>

      <DynamicDialogContainer
        isOpen={isOpen}
        onOpenChange={setIsOpen}
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
    </>
  );
};
