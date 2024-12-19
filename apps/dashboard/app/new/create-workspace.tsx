"use client";

import { Loading } from "@/components/dashboard/loading";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { Box } from "lucide-react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: z.string().trim().min(3, "Name is required and should be at least 3 characters").max(50),
});

export const CreateWorkspace: React.FC = () => {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const router = useRouter();
  const createWorkspace = trpc.workspace.create.useMutation({
    onSuccess: async ({ workspace }) => {
      toast.success("Your workspace has been created");
      router.push(`/new?workspaceId=${workspace.id}`);
    },
  });

  function AsideContent() {
    return (
      <div className="space-y-2">
        <div className="bg-primary/5 inline-flex items-center justify-center rounded-full border p-4">
          <Box className="text-primary h-6 w-6" />
        </div>
        <h2 className="text-lg font-medium">What is a workspace?</h2>
        <p className="text-content-subtle text-sm">
          A workspace groups all your resources and billing. You can have one personal workspace for
          free and create more workspaces with your teammates.
        </p>
      </div>
    );
  }
  return (
    <div className="flex flex-col md:flex-row items-start justify-between gap-8 md:gap-16">
      <main className="w-full md:w-3/4">
        <Form {...form}>
          <form
            onSubmit={form.handleSubmit((values) => createWorkspace.mutate({ ...values }))}
            className="flex flex-col space-y-4"
          >
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormMessage className="text-xs" />
                  <FormControl>
                    <Input {...field} />
                  </FormControl>
                  <FormDescription>What should your workspace be called?</FormDescription>
                </FormItem>
              )}
            />

            <div className="mt-8">
              <Button
                variant="primary"
                disabled={createWorkspace.isLoading || !form.formState.isValid}
                type="submit"
                className="w-full"
              >
                {createWorkspace.isLoading ? <Loading /> : "Create Workspace"}
              </Button>
            </div>
          </form>
        </Form>
      </main>
      <aside className="w-1/4 flex-col items-start justify-center space-y-16 max-md:hidden md:flex ">
        <AsideContent />
      </aside>
    </div>
  );
};
