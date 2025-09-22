"use client";
import { trpc } from "@/lib/trpc/client";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput, SettingCard, toast } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

export function UpdateWorkspaceName() {
  const workspace = useWorkspaceNavigation();
  const router = useRouter();
  const utils = trpc.useUtils();

  const [name, setName] = useState(workspace?.name);

  const formSchema = z.object({
    workspaceId: z.string(),
    workspaceName: z
      .string()
      .trim()
      .min(3, { message: "Workspace name must be at least 3 characters long" })
      .max(50, {
        message: "Workspace name must be less than 50 characters long",
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
    onSuccess() {
      toast.success("Workspace name updated");
      // invalidate the current user so it refetches
      utils.user.getCurrentUser.invalidate();
      utils.workspace.invalidate();
      setName(watch("workspaceName"));
      router.refresh();
    },
    onError(err) {
      toast.error("Failed to update workspace name", {
        description: err.message,
      });
    },
  });

  const onSubmit = async (values: z.infer<typeof formSchema>) => {
    if (workspace?.name === values.workspaceName || !values.workspaceName) {
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

  return (
    <form onSubmit={handleSubmit(onSubmit)} id="workspace-name-form">
      <SettingCard
        title={"Workspace Name"}
        description={
          "Not customer-facing. Choose a name that is easy to recognize."
        }
        border="top"
        className="border-b"
        contentWidth="w-full lg:w-[420px]"
      >
        <div className="flex flex-row justify-end items-center w-full gap-x-2">
          <input type="hidden" name="workspaceId" value={workspace?.id} />
          <label htmlFor="workspaceName" className="sr-only">
            Workspace Name
          </label>
          <FormInput
            className="w-[21rem]"
            placeholder="Workspace Name"
            minLength={3}
            maxLength={50}
            error={errors.workspaceName?.message}
            {...register("workspaceName")}
          />
          <Button
            type="submit"
            variant="primary"
            size="lg"
            disabled={
              updateName.isLoading ||
              isSubmitting ||
              !isValid ||
              watch("workspaceName") === name
            }
            loading={updateName.isLoading || isSubmitting}
          >
            Save
          </Button>
        </div>
      </SettingCard>
    </form>
  );
}
