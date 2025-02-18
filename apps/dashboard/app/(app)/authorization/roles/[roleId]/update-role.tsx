"use client";
import { revalidateTag } from "@/app/actions";
import { Loading } from "@/components/dashboard/loading";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
} from "@/components/ui/dialog";
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
import { tags } from "@/lib/cache";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { DialogTrigger } from "@radix-ui/react-dialog";
import type { Role } from "@unkey/db";
import { Button } from "@unkey/ui";
import { validation } from "@unkey/validation";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

type Props = {
  trigger: React.ReactNode;
  role: Role;
};

const formSchema = z.object({
  name: validation.name,
  description: validation.description.optional(),
});

export const UpdateRole: React.FC<Props> = ({ trigger, role }) => {
  const [open, setOpen] = useState(false);
  const router = useRouter();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      name: role.name,
      description: role.description ?? undefined,
    },
  });

  const updateRole = trpc.rbac.updateRole.useMutation({
    onMutate() {
      toast.loading("Updating Role");
    },
    onSuccess() {
      toast.success("Role updated");
      revalidateTag(tags.role(role.id));
      router.refresh();
      setOpen(false);
    },
    onError(err) {
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    updateRole.mutate({
      id: role.id,
      name: values.name,
      description: values.description ?? null,
    });
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Update Role</DialogTitle>
          <DialogDescription>
            Roles are used to group permissions together and are attached to keys.
          </DialogDescription>
        </DialogHeader>

        <Form {...form}>
          <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col gap-8">
            <FormField
              control={form.control}
              name="name"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Name</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="Manage domains and DNS records"
                      {...field}
                      className=" dark:focus:border-gray-700"
                    />
                  </FormControl>
                  <FormDescription>
                    A unique key to identify your role. We suggest using <code>.</code> (dot)
                    separated names, to structure your hierarchy. For example we use{" "}
                    <code>api.create_key</code> or <code>api.update_api</code> in our own
                    permissions.
                  </FormDescription>{" "}
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
                      placeholder="Perform CRUD operations for DNS records and domains."
                      {...field}
                      className=" dark:focus:border-gray-700"
                    />
                  </FormControl>
                  <FormDescription>Optionally explain what this role does.</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button type="submit">
                {updateRole.isLoading ? <Loading className="w-4 h-4" /> : "Save"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
