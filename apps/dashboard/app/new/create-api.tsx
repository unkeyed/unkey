"use client";

import { trpc } from "@/lib/trpc/client";
import { PostHogIdentify } from "@/providers/PostHogProvider";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput, toast } from "@unkey/ui";
import { Code2 } from "lucide-react";
import { useRouter } from "next/navigation";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: z.string().trim().min(3, "Name is required and should be at least 3 characters").max(50),
});

type Props = {
  workspace: {
    id: string;
    name: string;
  };
};

export const CreateApi: React.FC<Props> = ({ workspace }) => {
  const {
    handleSubmit,
    control,
    reset,
    formState: { errors, isValid },
  } = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const { data: user, isLoading } = trpc.user.getCurrentUser.useQuery();
  const router = useRouter();

  if (!isLoading && user) {
    PostHogIdentify({ user });
  }

  const createApi = trpc.api.create.useMutation({
    onSuccess: async ({ id: apiId }) => {
      toast.success("Your API has been created");
      reset();
      router.push(`/new?workspaceId=${workspace.id}&apiId=${apiId}`);
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
          <form
            onSubmit={handleSubmit((values) => createApi.mutate({ ...values }))}
            className="flex flex-col space-y-4"
          >
            <Controller
              control={control}
              name="name"
              render={({ field }) => (
                <div className="space-y-1.5 w-full">
                  <div className="text-gray-11 text-[13px] flex items-center">API Name</div>
                  <FormInput
                    {...field}
                    error={errors.name?.message}
                    description={
                      <>
                        <p>What should your api be called?</p>
                        <p>This is just for you, and will not be visible to your customers</p>
                      </>
                    }
                  />
                </div>
              )}
            />

            <div className="mt-8">
              <Button
                variant="primary"
                disabled={createApi.isLoading || !isValid}
                type="submit"
                loading={createApi.isLoading}
                className="w-full h-9"
              >
                Create API
              </Button>
            </div>
          </form>
        </div>
      </main>
      <aside className="flex-col items-start justify-center w-1/4 space-y-16 md:flex max-md:hidden ">
        <AsideContent />
      </aside>
    </div>
  );
};
