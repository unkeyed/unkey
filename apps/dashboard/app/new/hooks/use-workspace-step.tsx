import { setLastUsedOrgCookie, setSessionCookie } from "@/lib/auth/cookies";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { StackPerspective2 } from "@unkey/icons";
import { Button, FormInput, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useEffect, useRef, useState } from "react";
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
      "Use lowercase letters, numbers, and single hyphens (no leading/trailing hyphens).",
    ),
});

type WorkspaceFormData = z.infer<typeof workspaceSchema>;

type Props = {
  // Move to the next step
  advance: () => void;
};

export const useWorkspaceStep = (props: Props): OnboardingStep => {
  const [slugManuallyEdited, setSlugManuallyEdited] = useState(false);
  const [workspaceCreated, setWorkspaceCreated] = useState(false);
  const [isMounted, setIsMounted] = useState(false);
  const formRef = useRef<HTMLFormElement>(null);
  const router = useRouter();
  const utils = trpc.useUtils();

  useEffect(() => {
    setIsMounted(true);
  }, []);

  const form = useForm<WorkspaceFormData>({
    resolver: zodResolver(workspaceSchema),
    mode: "onChange",
    defaultValues: {
      workspaceName: "",
      slug: "",
    },
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
      await utils.stripe.invalidate();
      // Force a router refresh to ensure the server-side layout
      // re-renders with the new session context and fresh workspace data
      router.refresh();
    },
    onError: (error) => {
      toast.error(`Failed to load new workspace: ${error.message}`);
    },
  });

  const createWorkspace = trpc.workspace.create.useMutation({
    onSuccess: async ({ orgId }) => {
      setWorkspaceCreated(true);

      await switchOrgMutation.mutateAsync(orgId);
      try {
        await setLastUsedOrgCookie({ orgId });
      } catch (error) {
        console.error("Failed to persist last-used workspace:", error);
        // Continue anyway - cookie is a UX enhancement, not critical
      }

      props.advance();
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
      } else if (error.data?.code === "CONFLICT") {
        form.setError("slug", { message: error.message }, { shouldFocus: true });
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

  // Watch form values to ensure consistent hydration
  const workspaceName = form.watch("workspaceName");
  const slug = form.watch("slug");

  const validFieldCount = (["workspaceName", "slug"] as const).filter((fieldName) => {
    const hasError = Boolean(form.formState.errors[fieldName]);
    const value = fieldName === "workspaceName" ? workspaceName : slug;
    const hasValue = Boolean(value);
    return !hasError && hasValue;
  }).length;

  const isLoading = createWorkspace.isLoading;
  return {
    name: "Workspace",
    icon: <StackPerspective2 iconSize="sm-regular" className="text-gray-11" />,
    body: (
      <form ref={formRef} onSubmit={form.handleSubmit(onSubmit)}>
        <div className="flex flex-col">
          <div className="space-y-4 p-1">
            <FormInput
              {...form.register("workspaceName", {
                onChange: (evt) => {
                  // Re-validate on change to update validFieldCount
                  form.trigger("workspaceName");

                  // Only auto-generate if not manually edited
                  if (!slugManuallyEdited) {
                    if (evt.target.value.length >= 3) {
                      form.setValue("slug", slugify(evt.target.value), {
                        shouldValidate: true,
                      });
                    } else {
                      // Clear slug when workspace name is too short
                      form.setValue("slug", "", {
                        shouldValidate: false,
                      });
                    }
                  }
                },
              })}
              placeholder="Enter workspace name"
              label="Workspace name"
              required
              error={form.formState.errors.workspaceName?.message}
              disabled={isLoading || workspaceCreated}
            />
            <FormInput
              {...form.register("slug", {
                onChange: (evt) => {
                  // If we don't clear the manually set error, it will persist even if the user clears
                  // or changes the input
                  form.clearErrors("slug");
                  const v = evt.currentTarget.value;
                  setSlugManuallyEdited(v.length > 0);
                  form.setValue("slug", slugify(v), {
                    shouldValidate: true,
                  });
                  form.trigger("slug");
                },
              })}
              placeholder={isMounted ? "enter-a-handle" : ""}
              label="Workspace URL handle"
              required
              error={form.formState.errors.slug?.message}
              prefix="app.unkey.com/"
              maxLength={64}
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
    onStepNext: () => {
      if (workspaceCreated) {
        props.advance();
        return;
      }
      if (!isLoading) {
        formRef.current?.requestSubmit();
      }
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
