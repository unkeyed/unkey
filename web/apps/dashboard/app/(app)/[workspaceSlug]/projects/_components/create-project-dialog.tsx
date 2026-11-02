"use client";

import { routes } from "@/lib/navigation/routes";
import {
  type CreateProjectRequestSchema,
  createProjectRequestSchema,
  useCreateProject,
} from "@/lib/queries/projects";
import { slugify } from "@/lib/slugify";
import { zodResolver } from "@hookform/resolvers/zod";
import { ConflictErrorResponse } from "@unkey/api/models/errors";
import { Button, FormInput } from "@unkey/ui";
import dynamic from "next/dynamic";
import { useRouter } from "next/navigation";
import type React from "react";
import { useTransition } from "react";
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
  const [isNavigating, startNavigation] = useTransition();
  const createProject = useCreateProject();
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
      const projectId = await createProject.mutateAsync(values);
      // Keep the dialog (and its loading button) up until the new route has
      // rendered; closing immediately leaves the user staring at the old page.
      startNavigation(() => {
        router.push(routes.projects.apps.new({ workspaceSlug, projectId }));
      });
    } catch (error) {
      if (error instanceof ConflictErrorResponse) {
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
    <DynamicDialogContainer
      isOpen={isOpen}
      onOpenChange={(open) => {
        // Block escape/overlay/X while the project is being created or we're
        // navigating to it; closing would orphan the redirect.
        if (!open && isBusy) {
          return;
        }
        onOpenChange(open);
      }}
      title="Create New Project"
      footer={
        <div className="w-full flex flex-col gap-2 items-center justify-center">
          <Button
            type="submit"
            form="create-project-form"
            variant="primary"
            size="xlg"
            disabled={isBusy || !isValid}
            loading={isBusy}
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
