"use client";

import { Loading } from "@/components/dashboard/loading";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
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
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc/client";
import { useOrganizationList } from "@clerk/nextjs";
import { zodResolver } from "@hookform/resolvers/zod";
import { Box } from "lucide-react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: z.string().min(3, "Name is required and should be at least 3 characters").max(50),
});

export const CreateWorkspace: React.FC = () => {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const { setActive } = useOrganizationList();

  const { toast } = useToast();
  const router = useRouter();
  const createWorkspace = trpc.workspace.create.useMutation({
    onSuccess: async ({ workspace, organizationId }) => {
      toast({
        title: "Workspace Created",
        description: "Your workspace has been created",
      });

      if (setActive) {
        await setActive({ organization: organizationId });
      }
      router.push(`/new?workspaceId=${workspace.id}`);
    },
    onError(_err) {
      toast({
        title: "Error",
        description: "An error occured while creating your workspace, please contact support.",
        variant: "alert",
      });
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
    <div className="flex items-start justify-between gap-16">
      <main className="w-3/4">
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
            <div className="flex cursor-default items-start justify-between space-x-3 space-y-0 rounded-md border p-4">
              <p className="text-content-subtle text-sm">
                250 Monthly active keys and 10,000 verifications included.
              </p>
              <Badge>14 Day Trial</Badge>
            </div>
            <div className="mt-8">
              <Button
                variant={form.formState.isValid ? "primary" : "disabled"}
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
