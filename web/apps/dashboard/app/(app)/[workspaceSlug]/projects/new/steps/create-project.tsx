"use client";
import { collection } from "@/lib/collections";
import {
  type CreateProjectRequestSchema,
  createProjectRequestSchema,
} from "@/lib/collections/deploy/projects";
import { slugify } from "@/lib/slugify";
import { zodResolver } from "@hookform/resolvers/zod";
import { DuplicateKeyError } from "@tanstack/react-db";
import { Button, FormInput, useStepWizard } from "@unkey/ui";
import { useForm } from "react-hook-form";
import { OnboardingLinks } from "../onboarding-links";

type CreateProjectStepProps = {
  onProjectCreated: (id: string) => void;
};

export const CreateProjectStep = ({ onProjectCreated }: CreateProjectStepProps) => {
  const { next } = useStepWizard();

  const {
    register,
    handleSubmit,
    setValue,
    setError,
    formState: { errors, isSubmitting, isValid },
  } = useForm<CreateProjectRequestSchema>({
    resolver: zodResolver(createProjectRequestSchema),
    defaultValues: {
      name: "",
      slug: "",
    },
    mode: "onChange",
  });

  const onSubmitForm = async (values: CreateProjectRequestSchema) => {
    try {
      const tx = collection.projects.insert({
        name: values.name,
        slug: values.slug,
        repositoryFullName: null,
        currentDeploymentId: null,
        isRolledBack: false,
        id: "will-be-replace-by-server",
        latestDeploymentId: null,
        author: "will-be-replace-by-server",
        authorAvatar: "will-be-replace-by-server",
        branch: "will-be-replace-by-server",
        commitTimestamp: Date.now(),
        commitTitle: "will-be-replace-by-server",
        domain: "will-be-replace-by-server",
      });
      await tx.isPersisted.promise;
      onProjectCreated((tx.metadata as { projectId: string }).projectId);
      next();
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
  };

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setValue("slug", slugify(e.target.value));
  };

  return (
    <div className="w-full justify-center items-center flex flex-col">
      <div className="flex flex-col items-center border border-grayA-5 rounded-[14px] justify-center gap-4 py-[18px] px-4 min-w-[600px]">
        <form onSubmit={handleSubmit(onSubmitForm)} className="flex flex-col gap-4 w-full">
          <FormInput
            requirement="required"
            label="Project Name"
            className="[&_input:first-of-type]:h-[36px]"
            description="A descriptive name for your project."
            data-1p-ignore
            error={errors.name?.message}
            {...register("name", {
              onChange: handleNameChange,
            })}
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

          <Button
            type="submit"
            variant="primary"
            size="xlg"
            disabled={isSubmitting || !isValid}
            loading={isSubmitting}
            className="w-full rounded-lg mt-2"
          >
            Create Project
          </Button>
        </form>
      </div>
      <div className="mb-7" />
      <OnboardingLinks />
    </div>
  );
};
