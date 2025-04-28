"use client";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, Input, SettingCard } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  workspaceId: z.string(),
  workspaceName: z.string().trim(),
});

type Props = {
  workspace: {
    id: string;
    name: string;
  };
};

export const UpdateWorkspaceName: React.FC<Props> = ({ workspace }) => {
  const router = useRouter();
  const utils = trpc.useUtils();
  const [name, setName] = useState(workspace.name);
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      workspaceId: workspace.id,
      workspaceName: name,
    },
  });

  const updateName = trpc.workspace.updateName.useMutation({
    onSuccess() {
      toast.success("Workspace name updated");
      // invalidate the current user so it refetches
      utils.user.getCurrentUser.invalidate();
      setName(form.getValues("workspaceName"));
      router.refresh();
    },
    onError(err) {
      toast.error("Failed to update namespace name", {
        description: err.message,
      });
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (name === values.workspaceName || !values.workspaceName) {
      return toast.error("Please provide a different name before saving.");
    }

    await updateName.mutateAsync({ workspaceId: workspace.id, name: values.workspaceName });
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <SettingCard
          title={
            <div className="flex items-center justify-start gap-2.5">
              <span className="text-sm font-medium text-accent-12">Workspace Name</span>
            </div>
          }
          description={
            <div className="font-normal text-[13px] max-w-[380px]">
              Not customer-facing. Choose a name that is easy to recognize.
            </div>
          }
          border="top"
          contentWidth="w-full lg:w-[320px]"
        >
          <div className="flex flex-row justify-items-stretch items-center w-full gap-x-2">
            <input type="hidden" name="workspaceId" value={workspace.id} />
            <label htmlFor="workspaceName" className="hidden sr-only">
              Workspace Name
            </label>
            <FormField
              control={form.control}
              name="workspaceName"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Input
                      type="text"
                      id="workspaceName"
                      disabled={form?.formState?.isSubmitting || form.formState.isLoading}
                      className="w-[20rem] lg:w-[16rem]"
                      {...field}
                      autoComplete="off"
                      onBlur={(e) => {
                        if (e.target.value === "") {
                          return;
                        }
                      }}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
            <Button
              type="submit"
              size="lg"
              loading={form?.formState?.isSubmitting}
              disabled={
                !form.formState.isValid ||
                form.formState.isSubmitting ||
                name === form.watch("workspaceName")
              }
            >
              Save
            </Button>
          </div>
        </SettingCard>
      </form>
    </Form>
  );
};
