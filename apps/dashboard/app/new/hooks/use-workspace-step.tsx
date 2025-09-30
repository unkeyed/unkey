import { setSessionCookie } from "@/lib/auth/cookies";
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
  slug: z
    .string()
    .trim()
    .min(3, "Workspace slug must be at least 3 characters")
    .max(64, "Workspace slug must be 64 characters or less")
    .regex(
      /^[a-z0-9]+(?:-[a-z0-9]+)*$/,
      "Use lowercase letters, numbers, and single hyphens (no leading/trailing hyphens)."
    ),
});

type WorkspaceFormData = z.infer<typeof workspaceSchema>;

export const useWorkspaceStep = (): OnboardingStep => {
  const [slugManuallyEdited, setSlugManuallyEdited] = useState(false);
  const [workspaceCreated, setWorkspaceCreated] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);
  const router = useRouter();
  const utils = trpc.useUtils();

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

      await setSessionCookie({
        token: sessionData.token,
        expiresAt: sessionData.expiresAt,
      });
      // invalidate the user cache and workspace cache.
      await utils.user.getCurrentUser.invalidate();
      await utils.workspace.getCurrent.invalidate();
      await utils.api.invalidate();
      await utils.ratelimit.invalidate();
      // Force a router refresh to ensure the server-side layout
      // re-renders with the new session context and fresh workspace data
      router.refresh();
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
      slug: data.slug.toLowerCase(),
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
    icon: <StackPerspective2 iconsize="sm-regular" className="text-gray-11" />,
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
          {/*           <Refresh3 className="text-gray-12 !w-3 !h-3 flex-shrink-0" iconsize="sm-regular" /> */}
          {/*           Upload */}
          {/*         </div> */}
          {/*       </Button> */}
          {/*       <Button variant="outline" className="w-fit"> */}
          {/*         <div className="gap-2 flex items-center text-[13px] leading-4 font-medium"> */}
          {/*           <Refresh3 className="text-gray-12 !w-3 !h-3 flex-shrink-0" iconsize="sm-regular" /> */}
          {/*           Gradient */}
          {/*         </div> */}
          {/*       </Button> */}
          {/*       <Trash iconsize="md-medium" className="text-gray-8 ml-[10px]" /> */}
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
                const currentSlug = form.getValues("slug");
                const isSlugDirty = form.formState.dirtyFields.slug;

                // Only auto-generate if slug is empty, not dirty, and hasn't been manually edited
                if (!currentSlug && !isSlugDirty && !slugManuallyEdited) {
                  form.setValue("slug", slugify(evt.currentTarget.value), {
                    shouldValidate: true,
                  });
                }
              }}
              required
              error={form.formState.errors.workspaceName?.message}
              disabled={isLoading || workspaceCreated}
            />
            <FormInput
              {...form.register("slug")}
              placeholder="enter-a-handle"
              label="Workspace URL handle"
              required
              error={form.formState.errors.slug?.message}
              prefix="app.unkey.com/"
              maxLength={64}
              onChange={(evt) => {
                const v = evt.currentTarget.value;
                setSlugManuallyEdited(v.length > 0);
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
    .replace(/[^a-z0-9\s-]/g, "") // Remove special chars except lowercase letters, numbers, spaces, and hyphens
    .replace(/\s+/g, "-") // Replace spaces with hyphens
    .replace(/-+/g, "-") // Replace multiple hyphens with single
    .replace(/^-|-$/g, ""); // Remove leading/trailing hyphens
};
