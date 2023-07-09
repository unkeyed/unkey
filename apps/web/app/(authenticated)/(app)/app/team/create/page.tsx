"use client";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/Form";
import { Loading } from "@/components/loading";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useOrganizationList } from "@clerk/nextjs";
const formSchema = z.object({
  name: z.string().min(3).max(50),
  slug: z.string().min(1).max(50).regex(/^[a-zA-Z0-9-_\.]+$/),
});

export default function TeamCreation() {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const { createOrganization, isLoaded, setActive } = useOrganizationList();

  const { toast } = useToast();
  const router = useRouter();
  const createWorkspace = trpc.workspace.create.useMutation({
    onSuccess() {
      router.push("/app/stripe");
    },
    onError(err) {
      console.error(err);
      toast({ title: "Error", description: err.message, variant: "destructive" });
    },
  });
  if (!isLoaded) {
    return null;
  }

  const createClerkOrg = async ({ values }: { values: { name: string; slug: string } }) => {
    await createOrganization({
      ...values,
    })
      .then((res) => {
        setActive({ organization: res.id });
        createWorkspace.mutate({ ...values, tenantId: res.id });
      })
      .catch((err) => {
        console.error(err);
        toast({ title: "Error", description: err.message, variant: "destructive" });
      });
  };

  return (
    <div className="flex items-center justify-center w-full min-h-screen">
      <Card>
        <CardHeader>
          <CardTitle>Create your Team workspace</CardTitle>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form
              onSubmit={form.handleSubmit((values) => createClerkOrg({ values }))}
              className="flex flex-col space-y-4"
            >
              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Name</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormDescription>What should your team be called?</FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <FormField
                control={form.control}
                name="slug"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Slug</FormLabel>
                    <FormControl>
                      <Input {...field} />
                    </FormControl>
                    <FormDescription>
                      This will be used in urls etc. Only alphanumeric, dashes, underscores and
                      periods are allowed
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />

              <div className="mt-8">
                <Button
                  disabled={createWorkspace.isLoading || !form.formState.isValid}
                  type="submit"
                  className="w-full"
                >
                  {createWorkspace.isLoading ? <Loading /> : "Create"}
                </Button>
              </div>
            </form>
          </Form>
        </CardContent>
      </Card>
    </div>
  );
}
