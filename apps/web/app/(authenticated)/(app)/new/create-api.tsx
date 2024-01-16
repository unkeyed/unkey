"use client";

import { Loading } from "@/components/dashboard/loading";
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
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { PostHogIdentify } from "@/providers/PostHogProvider";
import { useUser } from "@clerk/nextjs";
import { zodResolver } from "@hookform/resolvers/zod";
import { Code2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
const formSchema = z.object({
  name: z.string().min(3, "Name is required and should be at least 3 characters").max(50),
});

type Props = {
  workspace: {
    id: string;
    name: string;
  };
};

export const CreateApi: React.FC<Props> = ({ workspace }) => {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const { user, isLoaded } = useUser();
  const router = useRouter();

  if (isLoaded && user) {
    PostHogIdentify({ user });
  }
  const createApi = trpc.api.create.useMutation({
    onSuccess: async ({ id: apiId }) => {
      toast.success("Your API has been created");

      router.push(`/new?workspaceId=${workspace.id}&apiId=${apiId}`);
    },
    onError(err) {
      toast.error(err.message);
    },
  });
  function AsideContent() {
    return (
      <div className="space-y-2">
        <div className="inline-flex items-center justify-center p-4 border rounded-full bg-primary/5">
          <Code2 className="w-6 h-6 text-primary" />
        </div>
        <h4 className="text-lg font-medium">What is an API?</h4>
        <p className="text-sm text-content-subtle">
          An API groups all of your keys together. They are invisible to your users but allow you to
          filter keys by a namespace. We recommend creating one API for each environment, typically{" "}
          <span className="font-medium text-mono text-foreground">development</span>,{" "}
          <span className="font-medium text-mono text-foreground">preview</span> and{" "}
          <span className="font-medium text-mono text-foreground">production</span>.
        </p>
        <ol className="ml-2 space-y-1 text-sm list-disc list-outside text-content-subtle">
          <li>Group keys together</li>
          <li>Globally distributed in 35+ locations</li>
          <li>Key and API analytics </li>
        </ol>
      </div>
    );
  }
  return (
    <div className="flex items-start justify-between gap-16">
      <main className="max-sm:w-full md:w-3/4">
        <aside className="mb-4 md:hidden">
          <AsideContent />
        </aside>

        <div>
          <Form {...form}>
            <form
              onSubmit={form.handleSubmit((values) => createApi.mutate({ ...values }))}
              className="flex flex-col space-y-4"
            >
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem className="w-full">
                    <FormLabel>API Name</FormLabel>
                    <FormMessage className="text-xs" />
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormDescription>
                      <p>What should your api be called?</p>
                      <p>This is just for you, and will not be visible to your customers</p>
                    </FormDescription>
                  </FormItem>
                )}
              />

              <div className="mt-8">
                <Button
                  variant={form.formState.isValid ? "primary" : "disabled"}
                  disabled={createApi.isLoading || !form.formState.isValid}
                  type="submit"
                  className="w-full"
                >
                  {createApi.isLoading ? <Loading /> : "Create API"}
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
