"use client";
import { cn } from "@/lib/utils";
import { Check } from "lucide-react";
import { Button } from "@/components/ui/button";
import { Particles } from "@/components/particles";
import { Input } from "@/components/ui/input";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc/client";
import {
  CardDescription,
  CardHeader,
  CardTitle,
  Card,
  CardContent,
  CardFooter,
} from "@/components/ui/card";
import { z } from "zod";
import { zodResolver } from "@hookform/resolvers/zod";
import { Loading } from "@/components/loading";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/Form";
import { useForm } from "react-hook-form";
import { useRouter } from "next/navigation";
const formSchema = z.object({
  name: z.string().min(3).max(50),
  slug: z.string().min(1).max(50).regex(/^[a-zA-Z0-9-_\.]+$/),
});

type Props = {
  workspaceId: string;
};
export const Onboarding: React.FC<Props> = ({ workspaceId }) => {
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
      toast({ title: "Error", description: err.message, variant: "destructive" });
    },
  });

  return (
    <div className="flex items-center justify-center min-h-screen w-full">
      <Card>
        <CardHeader>
          <CardTitle>Create your Workspace</CardTitle>
        </CardHeader>
        <CardContent>
          <Form {...form}>
            <form
              onSubmit={form.handleSubmit((values) => create.mutate({ ...values, workspaceId }))}
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
                    <FormDescription>What should your workspace be called?</FormDescription>
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
                <Button type="submit" className="w-full">
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
