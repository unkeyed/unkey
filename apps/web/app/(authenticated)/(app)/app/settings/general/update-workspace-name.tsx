"use client";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Form, FormControl, FormField, FormItem, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { useUser } from "@clerk/nextjs";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

export const dynamic = "force-dynamic";
const formSchema = z.object({
  workspaceId: z.string(),
  name: z.string(),
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
  const { user } = useUser();
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
  const updateName = trpc.workspace.changeName.useMutation({
    onSuccess() {
      toast.success("Workspace name updated");
      user?.reload();
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    updateName.mutateAsync(values);
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardHeader>
            <CardTitle>Workspace Name</CardTitle>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col space-y-2">
              <label className="sr-only hidden">Name</label>
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

              <p className="text-content-subtle text-xs">What should your workspace be called?</p>
            </div>
          </CardContent>
          <CardFooter className="justify-end">
            <Button
              variant={updateName.isLoading ? "disabled" : "primary"}
              type="submit"
              disabled={updateName.isLoading}
            >
              {updateName.isLoading ? <Loading /> : "Save"}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
