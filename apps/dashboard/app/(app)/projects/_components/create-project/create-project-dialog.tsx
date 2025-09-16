"use client";
import { NavbarActionButton } from "@/components/navigation/action-button";
import { Navbar } from "@/components/navigation/navbar";
import { collection } from "@/lib/collections";
import {
  type CreateProjectRequestSchema,
  createProjectRequestSchema,
} from "@/lib/collections/deploy/projects";
import { zodResolver } from "@hookform/resolvers/zod";
import { DuplicateKeyError } from "@tanstack/react-db";
import { Plus } from "@unkey/icons";
import { Button, DialogContainer, FormInput } from "@unkey/ui";
import { useState } from "react";
import { useForm } from "react-hook-form";

export const CreateProjectDialog = () => {
  const [isModalOpen, setIsModalOpen] = useState(false);

  const {
    register,
    handleSubmit,
    setValue,
    setError,
    reset,
    formState: { errors, isSubmitting, isValid },
  } = useForm<CreateProjectRequestSchema>({
    resolver: zodResolver(createProjectRequestSchema),
    defaultValues: {
      name: "",
      slug: "",
      gitRepositoryUrl: "",
    },
    mode: "onChange",
  });

  const onSubmitForm = async (values: CreateProjectRequestSchema) => {
    try {
      const tx = collection.projects.insert({
        name: values.name,
        slug: values.slug,
        gitRepositoryUrl: values.gitRepositoryUrl || null,
        activeDeploymentId: null,
        updatedAt: null,
        id: "will-be-replace-by-server",
      });
      await tx.isPersisted.promise;

      reset();
      setIsModalOpen(false);
    } catch (error) {
      if (error instanceof DuplicateKeyError) {
        setError("slug", {
          type: "custom",
          message: "Project with this slug already exists",
        });
      } else {
        console.error("Form submission error:", error);
        // The collection's onInsert will handle showing error toasts
      }
    }
  };

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    const name = e.target.value;
    const slug = name
      .toLowerCase()
      .replace(/[^a-z0-9\s-]/g, "")
      .replace(/\s+/g, "-")
      .replace(/-+/g, "-")
      .replace(/^-|-$/g, "");
    setValue("slug", slug);
  };

  const handleModalClose = (open: boolean) => {
    if (!open) {
      reset();
    }
    setIsModalOpen(open);
  };

  return (
    <>
      <Navbar.Actions>
        <NavbarActionButton title="Create new project" onClick={() => setIsModalOpen(true)}>
          <Plus />
          Create new project
        </NavbarActionButton>
      </Navbar.Actions>

      <DialogContainer
        isOpen={isModalOpen}
        onOpenChange={handleModalClose}
        title="Create New Project"
        subTitle="Set up a new project with a unique name and optional Git repository"
        footer={
          <div className="flex flex-col items-center justify-center w-full gap-2">
            <Button
              type="submit"
              form="project-form"
              variant="primary"
              size="xlg"
              disabled={isSubmitting || !isValid}
              loading={isSubmitting}
              className="w-full rounded-lg"
            >
              Create Project
            </Button>
            <div className="text-xs text-gray-9">
              Project will be available immediately after creation
            </div>
          </div>
        }
      >
        <form
          id="project-form"
          onSubmit={handleSubmit(onSubmitForm)}
          className="flex flex-col gap-4"
        >
          <FormInput
            required
            label="Project Name"
            className="[&_input:first-of-type]:h-[36px]"
            description="A descriptive name for your project."
            error={errors.name?.message}
            {...register("name", {
              onChange: handleNameChange,
            })}
            placeholder="My Awesome Project"
          />

          <FormInput
            required
            label="Slug"
            className="[&_input:first-of-type]:h-[36px]"
            description="URL-friendly identifier for your project (auto-generated from name)."
            error={errors.slug?.message}
            {...register("slug")}
            placeholder="my-awesome-project"
          />

          <FormInput
            label="Git Repository URL"
            className="[&_input:first-of-type]:h-[36px]"
            description="Optional: Link to your project's Git repository."
            error={errors.gitRepositoryUrl?.message}
            {...register("gitRepositoryUrl")}
            placeholder="https://github.com/username/repo"
          />
        </form>
      </DialogContainer>
    </>
  );
};
