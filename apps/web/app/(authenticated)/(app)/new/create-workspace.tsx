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
import { cn } from "@/lib/utils";
import { useOrganizationList } from "@clerk/nextjs";
import { zodResolver } from "@hookform/resolvers/zod";
import { Workspace } from "@unkey/db";
import { Box } from "lucide-react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: z.string().min(3, "Name is required and should be at least 3 characters").max(50),
  plan: z.enum(["free", "pro"]),
});

type Props = {
  workspaces: Workspace[];
};

export const CreateWorkspace: React.FC<Props> = ({ workspaces }) => {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      plan: workspaces.length === 0 ? "free" : "pro",
    },
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
        <div className="inline-flex items-center justify-center p-4 border rounded-full bg-primary/5">
          <Box className="w-6 h-6 text-primary" />
        </div>
        <h4 className="text-lg font-medium">What is a workspace?</h4>
        <p className="text-sm text-content-subtle">
          A workspace groups all your resources and billing. You can have one personal workspace for
          free and create more workspaces with your teammates.
        </p>
      </div>
    );
  }
  return (
    <div className="flex items-start justify-between gap-16 max-sm:gap-4">
      <main className="w-3/4 max-sm:w-full">
        <aside className="mb-4 md:hidden">
          <AsideContent />
        </aside>
        <div>
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
                <FormField
                  control={form.control}
                  name="plan"
                  render={({ field }) => (
                    <FormItem className="space-y-4">
                      <FormLabel className="text-lg">Plan</FormLabel>
                      <FormControl>
                        <div className="flex flex-col space-y-4">
                          <FormItem
                            className={cn(
                              "flex items-start justify-between space-x-3 space-y-0 border rounded-md p-4",
                              {
                                "border-primary": field.value === "pro",
                              },
                            )}
                            onClick={() => {
                              field.onChange("pro");
                            }}
                          >
                            <div className="w-full ">
                              <FormLabel className="font-semibold">PRO</FormLabel>

                              <p className="text-sm text-content-subtle max-sm:mt-2">
                                Usage based billing for teams
                              </p>
                              <p className="text-xs text-content-subtle">
                                250 Monthly active keys and 10,000 verifications included.
                              </p>
                            </div>

                            <Badge className="py-1 px-4 max-sm:text-xs whitespace-nowrap max-sm:absolute max-sm:right-12">
                              14 Day Trial
                            </Badge>
                          </FormItem>
                          <FormItem
                            className={cn(
                              "flex items-start justify-between  space-x-3 space-y-0 border rounded-md p-4",
                              {
                                "border-primary": field.value === "free",
                              },
                            )}
                            onClick={() => {
                              {
                                if (workspaces.length > 0) {
                                  // only one free workspace allowed
                                  return;
                                }
                                field.onChange("free");
                              }
                            }}
                          >
                            <div>
                              <div
                                className={cn({
                                  "opacity-60 cursor-disabled": workspaces.length > 0,
                                })}
                              >
                                <FormLabel className="font-semibold">FREE</FormLabel>
                                <p className="text-sm text-content-subtle max-sm:mt-2">
                                  Free forever for side projects
                                </p>
                                <p className="text-xs text-content-subtle">
                                  The free tier allows up to 100 monthly active keys and 2,500
                                  verifications per month.
                                </p>
                              </div>
                              {workspaces.length > 0 ? (
                                <p className="mt-2 text-xs text-alert">
                                  Only one free workspace allowed
                                </p>
                              ) : null}
                            </div>
                          </FormItem>
                        </div>
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
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
        </div>
      </main>
      <aside className="md:flex flex-col items-start justify-center w-1/4 space-y-16 max-md:hidden ">
        <AsideContent />
      </aside>
    </div>
  );
};
