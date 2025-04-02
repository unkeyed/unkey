"use client";
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
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

import { tags } from "@/lib/cache";
import { revalidateTag } from "../../../../actions";
const formSchema = z.object({
  name: z.string().trim().min(3, "Name is required and should be at least 3 characters"),
  apiId: z.string(),
  workspaceId: z.string(),
});

type Props = {
  api: {
    id: string;
    workspaceId: string;
    name: string;
  };
};

export const UpdateApiName: React.FC<Props> = ({ api }) => {
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: api.name,
      apiId: api.id,
      workspaceId: api.workspaceId,
    },
  });

  const updateName = trpc.api.updateName.useMutation({
    onSuccess() {
      toast.success("Your API name has been renamed!");
      revalidateTag(tags.api(api.id));
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });
  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (values.name === api.name || !values.name) {
      return toast.error("Please provide a valid name before saving.");
    }
    await updateName.mutateAsync(values);
  }

  return (
    <form onSubmit={form.handleSubmit(onSubmit)}>
      <Card>
        <CardHeader>
          <CardTitle>API Name</CardTitle>
          <CardDescription>
            API names are not customer facing. Choose a name that makes it easy to recognize for
            you.
          </CardDescription>
        </CardHeader>
        <CardContent>
          <div className="flex flex-col space-y-2">
            <input type="hidden" name="workspaceId" value={api.workspaceId} />
            <input type="hidden" name="apiId" value={api.id} />
            <label htmlFor="name" className="hidden sr-only">
              Name
            </label>
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => <Input className="max-w-sm" {...field} autoComplete="off" />}
            />
          </div>
        </CardContent>
        <CardFooter className="justify-end">
          <Button
            variant="primary"
            disabled={!form.formState.isValid || form.formState.isSubmitting}
            loading={form.formState.isSubmitting}
            type="submit"
          >
            Save
          </Button>
        </CardFooter>
      </Card>
    </form>
  );
};
