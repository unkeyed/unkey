"use client";
import { collection } from "@/lib/collections";
import { type CreateAppRequestSchema, createAppRequestSchema } from "@/lib/collections/deploy/apps";
import { zodResolver } from "@hookform/resolvers/zod";
import { DuplicateKeyError } from "@tanstack/react-db";
import { Button, FormInput, useStepWizard } from "@unkey/ui";
import { useForm } from "react-hook-form";
import { OnboardingLinks } from "../onboarding-links";

type CreateAppStepProps = {
  projectId: string;
  onAppCreated: (id: string) => void;
};

export const CreateAppStep = ({ projectId, onAppCreated }: CreateAppStepProps) => {
  const { next } = useStepWizard();

  const {
    register,
    handleSubmit,
    setValue,
    setError,
    formState: { errors, isSubmitting, isValid },
  } = useForm<CreateAppRequestSchema>({
    resolver: zodResolver(createAppRequestSchema),
    defaultValues: {
      projectId,
      name: "",
      slug: "",
    },
    mode: "onChange",
  });

  const onSubmitForm = async (values: CreateAppRequestSchema) => {
    try {
      const tx = collection.apps.insert({
        projectId: values.projectId,
        name: values.name,
        slug: values.slug,
        id: "will-be-replaced-by-server",
        defaultBranch: "main",
        currentDeploymentId: null,
        isRolledBack: false,
        repositoryFullName: null,
        latestDeploymentId: null,
        commitTitle: null,
        branch: "main",
        author: null,
        authorAvatar: null,
        commitTimestamp: null,
        domain: null,
      });
      await tx.isPersisted.promise;
      onAppCreated((tx.metadata as { appId: string }).appId);
      next();
    } catch (error) {
      if (error instanceof DuplicateKeyError) {
        setError("slug", {
          type: "custom",
          message: "App with this slug already exists",
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
            requirement="required"
            label="App Name"
            className="[&_input:first-of-type]:h-[36px]"
            description="A descriptive name for your app."
            data-1p-ignore
            error={errors.name?.message}
            {...register("name", {
              onChange: handleNameChange,
            })}
            placeholder="API Gateway"
          />

          <FormInput
            requirement="required"
            label="Slug"
            className="[&_input:first-of-type]:h-[36px]"
            description="URL-friendly identifier for your app (auto-generated from name)."
            data-1p-ignore
            error={errors.slug?.message}
            {...register("slug")}
            placeholder="api-gateway"
          />

          <Button
            type="submit"
            variant="primary"
            size="xlg"
            disabled={isSubmitting || !isValid}
            loading={isSubmitting}
            className="w-full rounded-lg mt-2"
          >
            Create App
          </Button>
        </form>
      </div>
      <div className="mb-7" />
      <OnboardingLinks />
    </div>
  );
};
