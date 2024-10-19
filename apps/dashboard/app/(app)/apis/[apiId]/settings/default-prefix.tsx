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
  keyAuthId: z.string(),
  workspaceId: z.string(),
  defaultPrefix: z.string(),
});

type Props = {
  keyAuth: {
    id: string;
    workspaceId: string;
    defaultPrefix: string | undefined | null;
  };
};

export const DefaultPrefix: React.FC<Props> = ({ keyAuth }) => {
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      defaultPrefix: keyAuth.defaultPrefix ?? undefined,
      keyAuthId: keyAuth.id,
      workspaceId: keyAuth.workspaceId,
    },
  });

  const setDefaultPrefix = trpc.api.setDefaultPrefix.useMutation({
    onSuccess() {
      toast.success("Default prefix for this API is updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });
  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (values.defaultPrefix.length > 8) {
      return toast.error("Default prefix is too long, maximum length is 8 characters.");
    }
    if (values.defaultPrefix === keyAuth.defaultPrefix) {
      return toast.error("Please provide a different prefix than already existing one as default");
    }
    await setDefaultPrefix.mutateAsync(values);
  }

  return (
    <form onSubmit={form.handleSubmit(onSubmit)}>
      <Card>
        <CardHeader>
          <CardTitle>Default Prefix</CardTitle>
          <CardDescription>Set default prefix for the keys under this API.</CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-2">
            <input type="hidden" name="workspaceId" value={keyAuth.workspaceId} />
            <input type="hidden" name="keyAuthId" value={keyAuth.id} />
            <label className="hidden sr-only">Default Prefix</label>
            <FormField
              control={form.control}
              name="defaultPrefix"
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
