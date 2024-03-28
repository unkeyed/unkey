"use client";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { FormField } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
const formSchema = z.object({
  name: z.string(),
  namespaceId: z.string(),
  workspaceId: z.string(),
});

type Props = {
  namespace: {
    id: string;
    workspaceId: string;
    name: string;
  };
};

export const UpdateNamespaceName: React.FC<Props> = ({ namespace }) => {
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: namespace.name,
      namespaceId: namespace.id,
      workspaceId: namespace.workspaceId,
    },
  });

  const updateName = trpc.ratelimit.namespace.update.name.useMutation({
    onSuccess() {
      toast.success("Your namespace name has been renamed!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });
  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (values.name === namespace.name || !values.name) {
      return toast.error("Please provide a valid name before saving.");
    }
    updateName.mutateAsync(values);
  }

  return (
    <form onSubmit={form.handleSubmit(onSubmit)}>
      <Card>
        <CardHeader>
          <CardTitle>Name</CardTitle>
          <CardDescription>
            Namespace names are not customer facing. Choose a name that makes it easy to recognize
            for you. Keep in mind this is used in your API calls, changing this might cause your
            ratelimit requests to get rejected.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-2">
            <label className="hidden sr-only">Name</label>
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => <Input className="max-w-sm" {...field} autoComplete="off" />}
            />
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button
            variant={
              form.formState.isValid && !form.formState.isSubmitting ? "primary" : "disabled"
            }
            disabled={!form.formState.isValid || form.formState.isSubmitting}
            type="submit"
          >
            {form.formState.isSubmitting ? <Loading /> : "Save"}
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
};
