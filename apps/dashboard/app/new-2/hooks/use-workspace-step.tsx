import { zodResolver } from "@hookform/resolvers/zod";
import { Refresh3, StackPerspective2, Trash } from "@unkey/icons";
import { Button, FormInput } from "@unkey/ui";
import { useRef, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import type { OnboardingStep } from "../components/onboarding-wizard";

const workspaceSchema = z.object({
  workspaceName: z
    .string()
    .trim()
    .min(3, "Workspace name is required")
    .max(50, "Workspace name must be 50 characters or less"),
  workspaceUrl: z
    .string()
    .min(3, "Workspace URL is required")
    .regex(
      /^[a-zA-Z0-9-_]+$/,
      "URL handle can only contain letters, numbers, hyphens, and underscores",
    ),
});

type WorkspaceFormData = z.infer<typeof workspaceSchema>;

export const useWorkspaceStep = (): OnboardingStep => {
  const [isSlugGenerated, setIsSlugGenerated] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);

  const form = useForm<WorkspaceFormData>({
    resolver: zodResolver(workspaceSchema),
    defaultValues: {
      workspaceName: "",
      workspaceUrl: "",
    },
    mode: "onChange",
  });

  const onSubmit = (data: WorkspaceFormData) => {
    console.info("Workspace form submitted:", data);
  };

  const validFieldCount = Object.keys(form.getValues()).filter((field) => {
    const fieldName = field as keyof WorkspaceFormData;
    const hasError = Boolean(form.formState.errors[fieldName]);
    const hasValue = Boolean(form.getValues(fieldName));
    return !hasError && hasValue;
  }).length;

  return {
    name: "Workspace",
    icon: <StackPerspective2 size="sm-regular" className="text-gray-11" />,
    body: (
      <form ref={formRef} onSubmit={form.handleSubmit(onSubmit)}>
        <div className="flex flex-col">
          <div className="flex flex-row py-1.5 gap-[18px]">
            <div className="bg-gradient-to-br from-info-5 to-blue-9 size-20 border rounded-2xl border-grayA-6" />
            <div className="flex flex-col gap-2">
              <div className="text-gray-11 text-[13px] leading-6">Company workspace logo</div>
              <div className="flex items-center gap-2">
                <Button variant="outline" className="w-fit">
                  <div className="gap-2 flex items-center text-[13px] leading-4 font-medium">
                    <Refresh3 className="text-gray-12 !w-3 !h-3 flex-shrink-0" size="sm-regular" />
                    Upload
                  </div>
                </Button>
                <Button variant="outline" className="w-fit">
                  <div className="gap-2 flex items-center text-[13px] leading-4 font-medium">
                    <Refresh3 className="text-gray-12 !w-3 !h-3 flex-shrink-0" size="sm-regular" />
                    Gradient
                  </div>
                </Button>
                <Trash size="md-regular" className="text-gray-8 ml-[10px]" />
              </div>
              <div className="text-gray-9 text-xs leading-6">
                .png, .jpg, or .svg up to 10MB, and 480×480px
              </div>
            </div>
          </div>
          <div className="space-y-4 pt-7">
            <FormInput
              {...form.register("workspaceName")}
              placeholder="Enter workspace name"
              label="Workspace name"
              onBlur={(evt) => {
                if (!isSlugGenerated) {
                  form.setValue("workspaceUrl", slugify(evt.currentTarget.value));
                  form.trigger("workspaceUrl");
                  setIsSlugGenerated(true);
                }
              }}
              required
              error={form.formState.errors.workspaceName?.message}
            />
            <FormInput
              {...form.register("workspaceUrl")}
              placeholder="enter-a-handle"
              label="Workspace URL handle"
              required
              error={form.formState.errors.workspaceUrl?.message}
              prefix="app.unkey.com/"
            />
          </div>
        </div>
      </form>
    ),
    kind: "required" as const,
    validFieldCount,
    requiredFieldCount: 2,
    buttonText: "Continue",
    description: "Set up your workspace to get started",
    onStepNext: () => {
      formRef.current?.requestSubmit();
    },
    onStepBack: () => {
      console.info("Going back from workspace step");
    },
  };
};

const slugify = (text: string): string => {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^\w\s-]/g, "") // Remove special chars except spaces and hyphens
    .replace(/\s+/g, "-") // Replace spaces with hyphens
    .replace(/-+/g, "-") // Replace multiple hyphens with single
    .replace(/^-|-$/g, ""); // Remove leading/trailing hyphens
};
