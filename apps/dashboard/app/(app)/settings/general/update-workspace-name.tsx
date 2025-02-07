"use client";
import { Loading } from "@/components/dashboard/loading";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormField, FormItem, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { useUser } from "@/lib/auth/hooks";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

export const dynamic = "force-dynamic";

const validCharactersRegex = /^[a-zA-Z0-9-_]+$/;

const formSchema = z.object({
  workspaceId: z.string(),
  name: z.string().trim().min(3),
});

type Props = {
  workspace: {
    id: string;
    tenantId: string;
    name: string;
  };
};

export const UpdateWorkspaceName: React.FC<Props> = ({ workspace }) => {
  const router = useRouter();
  const { fetchUser: refetchUser }  = useUser();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      workspaceId: workspace.id,
      name: workspace.name,
    },
  });
  const updateName = trpc.workspace.updateName.useMutation({
    onSuccess() {
      toast.success("Workspace name updated");
      refetchUser()
      router.refresh();
    },
    onError(err) {
      toast.error(err.message);
    },
  });

  function onSubmit(values) {
    updateName.mutateAsync(values);
  }
  const isDisabled = form.formState.isLoading || !form.formState.isValid || updateName.isLoading;
  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardHeader>
            <CardTitle>Workspace Name</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col space-y-2">
              <label className="hidden sr-only">Name</label>
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <Input {...field} className="max-w-sm" defaultValue={workspace.name} />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <p className="text-xs text-content-subtle">What should your workspace be called?</p>
            </div>
          </CardContent>
          <CardFooter className="justify-end">
            <Button variant="primary" type="submit" disabled={isDisabled}>
              {updateName.isLoading ? <Loading /> : "Save"}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
