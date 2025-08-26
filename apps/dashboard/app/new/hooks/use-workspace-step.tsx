import { setCookie } from "@/lib/auth/cookies";
import { UNKEY_SESSION_COOKIE } from "@/lib/auth/types";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { StackPerspective2 } from "@unkey/icons";
import { Button, FormInput, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
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
    .max(64, "Workspace URL must be 64 characters or less")
    .regex(
      /^(?![-])[a-zA-Z0-9-]+(?<![-])$/,
      "URL handle can only contain letters, numbers, and hyphens",
    ),
});

type WorkspaceFormData = z.infer<typeof workspaceSchema>;

export const useWorkspaceStep = (): OnboardingStep => {
  const [slugManuallyEdited, setSlugManuallyEdited] = useState(false);
  const [workspaceCreated, setWorkspaceCreated] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);
  const router = useRouter();

  const form = useForm<WorkspaceFormData>({
    resolver: zodResolver(workspaceSchema),
    mode: "onChange",
  });

  const switchOrgMutation = trpc.user.switchOrg.useMutation({
    onSuccess: async (sessionData) => {
      if (!sessionData.expiresAt) {
        console.error("Missing session data: ", sessionData);
        toast.error(`Failed to switch organizations: ${sessionData.error}`);
        return;
      }

      await setCookie({
        name: UNKEY_SESSION_COOKIE,
        value: sessionData.token,
        options: {
          httpOnly: true,
          secure: true,
          sameSite: "strict",
          path: "/",
          maxAge: Math.floor((sessionData.expiresAt.getTime() - Date.now()) / 1000),
        },
      });
    },
    onError: (error) => {
      toast.error(`Failed to load new workspace: ${error.message}`);
    },
  });

  const createWorkspace = trpc.workspace.create.useMutation({
    onSuccess: async ({ organizationId }) => {
      setWorkspaceCreated(true);
      switchOrgMutation.mutate(organizationId);
    },
    onError: (error) => {
      if (error.data?.code === "METHOD_NOT_SUPPORTED") {
        toast.error("", {
          style: {
            display: "flex",
            flexDirection: "column",
          },
          duration: 20000,
          description: error.message,
          action: (
            <div className="mx-auto pt-2">
              <Button
                onClick={() => {
                  toast.dismiss();
                  router.push("/apis");
                }}
              >
                Return to APIs
              </Button>
            </div>
          ),
        });
      } else {
        toast.error(`Failed to create workspace: ${error.message}`);
      }
    },
  });

  const onSubmit = async (data: WorkspaceFormData) => {
    if (workspaceCreated) {
      // Workspace already created, just proceed
      return;
    }
    createWorkspace.mutateAsync({
      name: data.workspaceName,
      slug: data.workspaceUrl.toLowerCase(),
    });
  };

  const validFieldCount = Object.keys(form.getValues()).filter((field) => {
    const fieldName = field as keyof WorkspaceFormData;
    const hasError = Boolean(form.formState.errors[fieldName]);
    const hasValue = Boolean(form.getValues(fieldName));
    return !hasError && hasValue;
  }).length;

  const isLoading = createWorkspace.isLoading;

  return {
    name: "Workspace",
    icon: <StackPerspective2 size="sm-regular" className="text-gray-11" />,
    body: (
      <form ref={formRef} onSubmit={form.handleSubmit(onSubmit)}>
        <div className="flex flex-col">
          {/* <div className="flex flex-row py-1.5 gap-[18px]"> */}
          {/*   <div className="bg-gradient-to-br from-info-5 to-blue-9 size-20 border rounded-2xl border-grayA-6" /> */}
          {/*   <div className="flex flex-col gap-2"> */}
          {/*     <div className="text-gray-11 text-[13px] leading-6">Company workspace logo</div> */}
          {/*     <div className="flex items-center gap-2"> */}
          {/*       <Button variant="outline" className="w-fit"> */}
          {/*         <div className="gap-2 flex items-center text-[13px] leading-4 font-medium"> */}
          {/*           <Refresh3 className="text-gray-12 !w-3 !h-3 flex-shrink-0" size="sm-regular" /> */}
          {/*           Upload */}
          {/*         </div> */}
          {/*       </Button> */}
          {/*       <Button variant="outline" className="w-fit"> */}
          {/*         <div className="gap-2 flex items-center text-[13px] leading-4 font-medium"> */}
          {/*           <Refresh3 className="text-gray-12 !w-3 !h-3 flex-shrink-0" size="sm-regular" /> */}
          {/*           Gradient */}
          {/*         </div> */}
          {/*       </Button> */}
          {/*       <Trash size="md-regular" className="text-gray-8 ml-[10px]" /> */}
          {/*     </div> */}
          {/*     <div className="text-gray-9 text-xs leading-6"> */}
          {/*       .png, .jpg, or .svg up to 10MB, and 480×480px */}
          {/*     </div> */}
          {/*   </div> */}
          {/* </div> */}

          {/* Use this 'pt-7' version when implementing profile photo and slug based onboarding*/}
          {/* <div className="space-y-4 pt-7"> */}
          <div className="space-y-4 p-1">
            <FormInput
              {...form.register("workspaceName")}
              placeholder="Enter workspace name"
              label="Workspace name"
              onBlur={(evt) => {
                const currentSlug = form.getValues("workspaceUrl");
                const isSlugDirty = form.formState.dirtyFields.workspaceUrl;

                // Only auto-generate if slug is empty, not dirty, and hasn't been manually edited
                if (!currentSlug && !isSlugDirty && !slugManuallyEdited) {
                  form.setValue("workspaceUrl", slugify(evt.currentTarget.value));
                  form.trigger("workspaceUrl");
                }
              }}
              required
              error={form.formState.errors.workspaceName?.message}
              disabled={isLoading || workspaceCreated}
            />
            <FormInput
              {...form.register("workspaceUrl")}
              placeholder="enter-a-handle"
              label="Workspace URL handle"
              required
              error={form.formState.errors.workspaceUrl?.message}
              prefix="app.unkey.com/"
              maxLength={64}
              onChange={(evt) => {
                // Mark slug as manually edited when user changes it
                if (evt.currentTarget.value) {
                  setSlugManuallyEdited(true);
                }
              }}
            />
          </div>
        </div>
      </form>
    ),
    kind: "required" as const,
    validFieldCount,
    requiredFieldCount: 2,
    buttonText: workspaceCreated ? "Continue" : "Create workspace",
    description: workspaceCreated
      ? "Workspace created successfully, continue to next step"
      : "Set up your workspace to get started",
    onStepNext: workspaceCreated
      ? undefined
      : () => {
          if (isLoading) {
            return;
          }

          formRef.current?.requestSubmit();
        },
    onStepBack: () => {
      console.info("Going back from workspace step");
    },
    isLoading,
  };
};

const slugify = (text: string): string => {
  return text
    .toLowerCase()
    .trim()
    .replace(/[^a-zA-Z0-9\s-]/g, "") // Remove special chars except letters, numbers, spaces, and hyphens
    .replace(/\s+/g, "-") // Replace spaces with hyphens
    .replace(/-+/g, "-") // Replace multiple hyphens with single
    .replace(/^-|-$/g, ""); // Remove leading/trailing hyphens
};
