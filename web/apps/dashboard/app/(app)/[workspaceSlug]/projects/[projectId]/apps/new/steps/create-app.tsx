"use client";
import { useDeployActionGate } from "@/app/(app)/[workspaceSlug]/projects/_components/hooks/use-deploy-action-gate";
import { createAppRequestSchema } from "@/lib/collections/deploy/apps";
import { slugify } from "@/lib/slugify";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput, useStepWizard } from "@unkey/ui";
import { useForm } from "react-hook-form";
import type { z } from "zod";
import { OnboardingLinks } from "../onboarding-links";

const formSchema = createAppRequestSchema.omit({ projectId: true, source: true });
export type AppDetails = z.infer<typeof formSchema>;

type CreateAppStepProps = {
  onAppDetailsSubmitted: (details: AppDetails) => void;
};

export const CreateAppStep = ({ onAppDetailsSubmitted }: CreateAppStepProps) => {
  const { next } = useStepWizard();
  const { gated, openPaywall, planGate } = useDeployActionGate();

  const {
    register,
    handleSubmit,
    setValue,
    formState: { errors, isSubmitting, isValid },
  } = useForm<AppDetails>({
    resolver: zodResolver(formSchema),
    defaultValues: { name: "", slug: "" },
    mode: "onChange",
  });

  const onSubmitForm = (values: AppDetails) => {
    if (gated) {
      openPaywall();
      return;
    }
    onAppDetailsSubmitted(values);
    next();
  };

  const handleNameChange = (e: React.ChangeEvent<HTMLInputElement>) => {
    setValue("slug", slugify(e.target.value));
  };

  return (
    <div className="w-full justify-center items-center flex flex-col">
      <div className="flex flex-col items-center border border-grayA-5 rounded-lg justify-center gap-4 py-[18px] px-4 min-w-[600px]">
        <form onSubmit={handleSubmit(onSubmitForm)} className="flex flex-col gap-4 w-full">
          <FormInput
            requirement="required"
            label="App Name"
            className="[&_input:first-of-type]:h-[36px]"
            autoFocus
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
            Continue
          </Button>
        </form>
      </div>
      <div className="mb-7" />
      <OnboardingLinks />
      {planGate}
    </div>
  );
};
