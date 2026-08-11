"use client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput, InfoTooltip, SettingCard, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

export function UpdateWorkspaceName() {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const utils = trpc.useUtils();

  // Server-side `requireWorkspaceAdmin` enforces this on the changeName
  // mutation; we mirror it on the client purely for UX so non-admin members
  // get a clear "admin required" affordance instead of a request that fails
  // with FORBIDDEN.
  const { data: currentUser } = trpc.user.getCurrentUser.useQuery();
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
  } = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "onChange",
    defaultValues: {
      workspaceId: workspace?.id,
      workspaceName: workspace?.name,
    },
  });

  // Force immediate refetch of all workspace-related queries
  const refetchWorkspaceName = async () => {
    await Promise.all([
      utils.user.getCurrentUser.refetch(),
      utils.workspace.getCurrent.refetch(),
      utils.user.listMemberships.refetch(),
    ]);
    router.refresh();
  };

  const updateName = trpc.workspace.updateName.useMutation({
    async onSuccess(data) {
      // updated: false means the server verified both sides already agree and
      // wrote nothing. The refetch still runs: this client's cache may be
      // exactly what is stale.
      if (data.updated) {
        toast.success("Workspace name updated");
      } else {
        toast.success("Workspace name is already up to date");
      }
      await refetchWorkspaceName();
    },
    async onError(err) {
      const code = err.data?.code;
      // PRECONDITION_FAILED means the name is saved but the auth provider has
      // not confirmed it, so the toast must not claim the update failed.
      if (code === "PRECONDITION_FAILED") {
        toast.warning("Workspace name not fully synced", {
          description: err.message,
        });
      } else {
        toast.error("Failed to update workspace name", {
          description: err.message,
        });
      }
      // Only these outcomes can leave the cached name stale; pure rejections
      // (validation, permissions, rate limits) change nothing.
      if (
        code === "PRECONDITION_FAILED" ||
        code === "INTERNAL_SERVER_ERROR" ||
        code === "CONFLICT"
      ) {
        await refetchWorkspaceName();
      }
    },
  });

  // Same-name submits are allowed on purpose: the server treats them as a
  // repair attempt that re-syncs the auth-provider org name.
  const onSubmit = async (values: z.infer<typeof formSchema>) => {
    if (!workspace?.id) {
      return toast.error("Workspace not found");
    }

    try {
      await updateName.mutateAsync({
        workspaceId: workspace.id,
        name: values.workspaceName,
      });
    } catch {
      // onError already surfaced the failure; catching keeps handleSubmit
      // from rethrowing it as an unhandled promise rejection.
    }
  };

  return (
    <form onSubmit={handleSubmit(onSubmit)} id="workspace-name-form">
      <SettingCard
        title={"Workspace Name"}
        description={"Not customer-facing. Choose a name that is easy to recognize."}
        border="top"
        className="border-b border-grayA-4"
        contentWidth="w-full lg:w-[420px]"
      >
        <div className="flex flex-row justify-end items-center w-full gap-x-2">
          <input type="hidden" name="workspaceId" value={workspace?.id} />
          <label htmlFor="workspaceName" className="sr-only">
            Workspace Name
          </label>
          <FormInput
            className="w-84"
            placeholder="Workspace Name"
            minLength={3}
            maxLength={50}
            error={errors.workspaceName?.message}
            {...register("workspaceName")}
          />
          <InfoTooltip
            content="Admin access required to rename the workspace"
            disabled={isAdmin}
            asChild
          >
            <span>
              <Button
                type="submit"
                variant="primary"
                size="lg"
                disabled={!isAdmin || updateName.isLoading || isSubmitting || !isValid}
                loading={updateName.isLoading || isSubmitting}
              >
                Save
              </Button>
            </span>
          </InfoTooltip>
        </div>
      </SettingCard>
    </form>
  );
}
