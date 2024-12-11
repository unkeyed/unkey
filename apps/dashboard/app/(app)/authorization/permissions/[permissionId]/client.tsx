"use client";

import { revalidateTag } from "@/app/actions";
import { Loading } from "@/components/dashboard/loading";
import { Card, CardContent, CardFooter } from "@/components/ui/card";
import { tags } from "@/lib/cache";
import { Button } from "@unkey/ui";

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Permission } from "@unkey/db";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

type Props = {
  permission: Permission;
  className?: string;
};

const formSchema = z.object({
  name: z.string(),
  description: z.string().optional(),
});

export const Client: React.FC<Props> = ({ permission }) => {
  const router = useRouter();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: permission.name,
      description: permission.description ?? undefined,
    },
  });

  const updatePermission = trpc.rbac.updatePermission.useMutation({
    onSuccess() {
      toast.success("Permission updated");
      revalidateTag(tags.permission(permission.id));
      router.refresh();
    },
    onError(err) {
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    updatePermission.mutate({
      id: permission.id,
      name: values.name,
      description: values.description ?? null,
    });
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardContent className="flex flex-col gap-8">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input placeholder="Manage domains and DNS records" {...field} />
                  </FormControl>
                  <FormDescription>
                    A unique key to identify your permission. We suggest using <code>.</code> (dot)
                    separated names, to structure your hierarchy. For example we use{" "}
                    <code>api.create_key</code> or <code>api.update_api</code> in our own
                    permissions.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />

            <FormField
              control={form.control}
              name="description"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Description</FormLabel>
                  <FormControl>
                    <Textarea
                      rows={form.getValues().description?.split("\n").length ?? 3}
                      placeholder="<Empty>"
                      {...field}
                    />
                  </FormControl>
                  <FormDescription>Optionally explain what this permission does.</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
          <CardFooter className="justify-end">
            <Button type="submit">
              {updatePermission.isLoading ? <Loading className="w-4 h-4" /> : "Save"}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
