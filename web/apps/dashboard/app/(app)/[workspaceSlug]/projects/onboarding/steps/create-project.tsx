"use client";
import { collection } from "@/lib/collections";
import {
  type CreateProjectRequestSchema,
  createProjectRequestSchema,
} from "@/lib/collections/deploy/projects";
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
        liveDeploymentId: null,
        isRolledBack: false,
        id: "will-be-replace-by-server",
        latestDeploymentId: null,
        author: "will-be-replace-by-server",
        authorAvatar: "will-be-replace-by-server",
        branch: "will-be-replace-by-server",
        commitTimestamp: Date.now(),
        commitTitle: "will-be-replace-by-server",
        domain: "will-be-replace-by-server",
        regions: [],
      });
      await tx.isPersisted.promise;
      // await collection.projects.utils.refetch();
      const created = collection.projects.toArray.find((p) => p.slug === values.slug);
      if (created) {
        onProjectCreated(created.id);
      }
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
    const name = e.target.value;
    const slug = name
      .toLowerCase()
      .replace(/[^a-z0-9\s-]/g, "")
      .replace(/\s+/g, "-")
      .replace(/-+/g, "-")
      .replace(/^-|-$/g, "");
    setValue("slug", slug);
  };

  return (
    <div className="w-full justify-center items-center flex flex-col">
      <div className="flex flex-col items-center border border-grayA-5 rounded-[14px] justify-center gap-4 py-[18px] px-4 min-w-[600px]">
        <form onSubmit={handleSubmit(onSubmitForm)} className="flex flex-col gap-4 w-full">
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
