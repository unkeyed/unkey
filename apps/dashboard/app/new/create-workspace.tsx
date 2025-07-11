"use client";

import { setCookie } from "@/lib/auth/cookies";
import { UNKEY_SESSION_COOKIE } from "@/lib/auth/types";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput, toast } from "@unkey/ui";
import { Box } from "lucide-react";
import { useRouter } from "next/navigation";
import { useRef, useTransition } from "react";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: z.string().trim().min(3, "Name is required and should be at least 3 characters").max(50),
});

export const CreateWorkspace: React.FC = () => {
  const {
    handleSubmit,
    control,
    formState: { errors, isValid },
  } = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const router = useRouter();
  const [isPending, startTransition] = useTransition();
  const workspaceIdRef = useRef<string | null>(null);

  const switchOrgMutation = trpc.user.switchOrg.useMutation({
    onSuccess: async (sessionData) => {
      if (!sessionData.expiresAt) {
        console.error("Missing session data: ", sessionData);
        toast.error(`Failed to switch organizations: ${sessionData.error}`);
        return;
      }

      await setCookie({
        name: UNKEY_SESSION_COOKIE,
        value: sessionData.token,
        options: {
          httpOnly: true,
          secure: true,
          sameSite: "strict",
          path: "/",
          maxAge: Math.floor((sessionData.expiresAt.getTime() - Date.now()) / 1000),
        },
      }).then(() => {
        startTransition(() => {
          router.push(`/new?workspaceId=${workspaceIdRef.current}`);
        });
      });
    },
    onError: (error) => {
      toast.error(`Failed to load new workspace: ${error.message}`);
    },
  });

  const createWorkspace = trpc.workspace.create.useMutation({
    onSuccess: async ({ workspace, organizationId }) => {
      workspaceIdRef.current = workspace.id;
      switchOrgMutation.mutate(organizationId);
    },
    onError: (error) => {
      if (error.data?.code === "METHOD_NOT_SUPPORTED") {
        toast.error("", {
          style: {
            display: "flex",
            flexDirection: "column",
          },
          duration: 20000,
          description: error.message,
          action: (
            <div className="mx-auto pt-2">
              <Button
                onClick={() => {
                  toast.dismiss();
                  router.push("/apis");
                }}
              >
                Return to APIs
              </Button>
            </div>
          ),
        });
      } else {
        toast.error(`Failed to create workspace: ${error.message}`);
      }
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
          A workspace groups all your resources and billing. You can create free workspaces for
          individual use, or upgrade to a paid workspace to collaborate with team members.
        </p>
      </div>
    );
  }

  return (
    <div className="flex flex-col md:flex-row items-start justify-between gap-8 md:gap-16">
      <main className="w-full md:w-3/4">
        <form
          onSubmit={handleSubmit((values) => createWorkspace.mutate({ ...values }))}
          className="flex flex-col space-y-4"
        >
          <Controller
            control={control}
            name="name"
            render={({ field }) => (
              <div className="space-y-1.5">
                <div className="text-gray-11 text-[13px] flex items-center">Name</div>
                <FormInput
                  {...field}
                  error={errors.name?.message}
                  description="What should your workspace be called?"
                />
              </div>
            )}
          />

          <div className="mt-8">
            <Button
              variant="primary"
              disabled={createWorkspace.isLoading || isPending || !isValid}
              type="submit"
              loading={createWorkspace.isLoading || isPending}
              className="w-full h-9"
            >
              Create Workspace
            </Button>
          </div>
        </form>
      </main>
      <aside className="w-1/4 flex-col items-start justify-center space-y-16 max-md:hidden md:flex ">
        <AsideContent />
      </aside>
    </div>
  );
};
