"use client";
import { collection } from "@/lib/collections";
import { createAppRequestSchema } from "@/lib/collections/deploy/apps";
import { zodResolver } from "@hookform/resolvers/zod";
import { DuplicateKeyError } from "@tanstack/react-db";
import { Button, FormInput, useStepWizard } from "@unkey/ui";
import { useForm } from "react-hook-form";
import type { z } from "zod";
import { OnboardingLinks } from "../onboarding-links";

const formSchema = createAppRequestSchema.omit({ projectId: true });
type FormValues = z.infer<typeof formSchema>;

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
  } = useForm<FormValues>({
    resolver: zodResolver(formSchema),
    defaultValues: { name: "", slug: "" },
    mode: "onChange",
  });

  const onSubmitForm = async (values: FormValues) => {
    try {
      const tx = collection.apps.insert({
        projectId,
        name: values.name,
        slug: values.slug,
        defaultBranch: "main",
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
    const slug = e.target.value
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
            {...register("name", { onChange: handleNameChange })}
            placeholder="My Awesome App"
          />

          <FormInput
            requirement="required"
            label="Slug"
            className="[&_input:first-of-type]:h-[36px]"
            description="URL-friendly identifier for your app (auto-generated from name)."
            data-1p-ignore
            error={errors.slug?.message}
            {...register("slug")}
            placeholder="my-awesome-app"
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
