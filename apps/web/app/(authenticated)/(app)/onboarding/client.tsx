"use client";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  name: z.string().min(3, "Name is required and should be at least 3 characters").max(50),
  slug: z.string().min(1, "Slug is required").max(50).regex(/^[a-zA-Z0-9-_\.]+$/),
});

type Props = {
  tenantId: string;
};
export const Onboarding: React.FC<Props> = ({ tenantId }) => {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const { toast } = useToast();
  const router = useRouter();
  const create = trpc.workspace.create.useMutation({
    onSuccess() {
      toast({
        title: "Workspace Created",
        description: "Your workspace has been created",
      });
      router.push("/app");
    },
    onError(err) {
      console.error(err);
      toast({
        title: "Error",
        description: err.message,
        variant: "destructive",
      });
    },
  });
  return (
    <div className="flex flex-col items-center justify-center w-full min-h-screen">
      <Card className="z-10 mx-6 shadow-md md:mx-0">
        <CardHeader>
          <CardTitle className="py-2 bg-gradient-to-tr border-zinc-400">
            <p className="text-sm ">Let's get started</p>
            Create your first Workspace
          </CardTitle>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form
              onSubmit={form.handleSubmit((values) => create.mutate({ ...values, tenantId }))}
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

              <FormField
                control={form.control}
                name="slug"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Slug</FormLabel>
                    <FormMessage className="text-xs" />
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormDescription>
                      This will be used in urls etc. Only alphanumeric, dashes, underscores and
                      periods are allowed
                    </FormDescription>
                  </FormItem>
                )}
              />

              <div className="mt-8">
                <Button
                  disabled={create.isLoading || !form.formState.isValid}
                  type="submit"
                  className="w-full"
                >
                  {create.isLoading ? <Loading /> : "Create"}
                </Button>
              </div>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  );
};
