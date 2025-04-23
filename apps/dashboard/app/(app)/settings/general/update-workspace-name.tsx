"use client";
import { SettingCard } from "@/components/settings-card";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Input } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

export const dynamic = "force-dynamic";

const formSchema = z.object({
  workspaceId: z.string(),
  workspaceName: z.string().trim().min(3),
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
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      workspaceId: workspace.id,
      workspaceName: workspace.name,
    },
  });
  const updateName = trpc.workspace.updateName.useMutation({
    onSuccess() {
      toast.success("Workspace name updated");
      // invalidate the current user so it refetches
      utils.user.getCurrentUser.invalidate();
      router.refresh();
    },
    onError(err) {
      toast.error(err.message);
    },
  });

  function onSubmit(values: z.infer<typeof formSchema>) {
    updateName.mutateAsync({ workspaceId: workspace.id, name: values.workspaceName });
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
                      className="w-[20rem] lg:w-[16rem] h-9"
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
              size="lg"
              variant="primary"
              disabled={
                !form.formState.isValid ||
                form.formState.isSubmitting ||
                workspace.name === form.watch("workspaceName")
              }
              loading={form.formState.isSubmitting}
              type="submit"
              className="justify-self-end"
            >
              Save
            </Button>
          </div>
        </SettingCard>
      </form>
    </Form>
  );
};
