"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { zodResolver } from "@hookform/resolvers/zod";
import { FormInput, SettingsForm, SettingsRow, formSaveState, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

export function UpdateWorkspaceName() {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const utils = trpc.useUtils();

  const [name, setName] = useState(workspace?.name);

  // Server-side `requireWorkspaceAdmin` enforces this on the changeName
  // mutation; we mirror it on the client purely for UX so non-admin members
  // get a clear "admin required" affordance instead of a request that fails
  // with FORBIDDEN.
  const { user: currentUser } = useWorkspace();
  const isAdmin = currentUser?.role === "admin";

  const formSchema = z.object({
    workspaceId: z.string(),
    workspaceName: z
      .string()
      .trim()
      .min(3, {
        error: "Workspace name must be at least 3 characters long",
      })
      .max(50, {
        error: "Workspace name must be less than 50 characters long",
      }),
  });

  const {
    register,
    handleSubmit,
    formState: { errors, isValid, isSubmitting },
    watch,
  } = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
    defaultValues: {
      workspaceId: workspace?.id,
      workspaceName: name,
    },
  });

  const updateName = trpc.workspace.updateName.useMutation({
    async onSuccess(_data, variables) {
      toast.success("Workspace name updated");
      // Force immediate refetch of all workspace-related queries
      await Promise.all([
        utils.workspace.getCurrent.refetch(),
        utils.user.listMemberships.refetch(),
        utils.workspace.listAvailable.invalidate(),
      ]);
      setName(variables.name);
      router.refresh();
    },
    onError(err) {
      toast.error("Failed to update workspace name", {
        description: err.message,
      });
    },
  });

  const isUnchanged = (value: string | undefined) => (value ?? "").trim() === name.trim();

  const onSubmit = async (values: z.infer<typeof formSchema>) => {
    if (isUnchanged(values.workspaceName) || !values.workspaceName) {
      return toast.error("Please provide a different name before saving.");
    }

    if (!workspace?.id) {
      return toast.error("Workspace not found");
    }

    await updateName.mutateAsync({
      workspaceId: workspace.id,
      name: values.workspaceName,
    });
  };

  const adminRequired = isAdmin ? undefined : "Admin access required to rename the workspace";
  const isDirty = !isUnchanged(watch("workspaceName"));
  const saveState = formSaveState({
    isSubmitting: updateName.isLoading || isSubmitting,
    isValid,
    isDirty,
    blockedReason: adminRequired,
  });

  return (
    <SettingsForm dirty={isDirty} onSubmit={handleSubmit(onSubmit)} saveState={saveState}>
      <SettingsRow
        title="Workspace Name"
        description="Not customer-facing. Choose a name that is easy to recognize."
      >
        <input type="hidden" name="workspaceId" value={workspace?.id} />
        <FormInput
          aria-label="Workspace Name"
          placeholder="Workspace Name"
          minLength={3}
          maxLength={50}
          description={adminRequired}
          error={errors.workspaceName?.message}
          {...register("workspaceName")}
          disabled={!isAdmin}
        />
      </SettingsRow>
    </SettingsForm>
  );
}
