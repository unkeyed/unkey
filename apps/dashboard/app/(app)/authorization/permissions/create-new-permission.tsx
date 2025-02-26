"use client";

import { Loading } from "@/components/dashboard/loading";
import { Badge } from "@/components/ui/badge";
import { Button } from "@unkey/ui";

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
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { DialogTrigger } from "@radix-ui/react-dialog";
import { validation } from "@unkey/validation";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
type Props = {
  trigger: React.ReactNode;
};

const formSchema = z.object({
  name: validation.name,
  description: validation.description,
});

export const CreateNewPermission: React.FC<Props> = ({ trigger }) => {
  const router = useRouter();
  const [open, setOpen] = useState(false);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const createPermission = trpc.rbac.createPermission.useMutation({
    onSuccess() {
      toast.success("Permission created");
      router.refresh();
      form.reset({
        name: "",
        description: "",
      });
      setOpen(false);
    },
    onError(err) {
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (values.description === "") {
      delete values.description;
    }
    createPermission.mutate(values);
  }
  function handleDialogOpenChange(newState: boolean) {
    setOpen(newState);
    form.reset();
  }

  return (
    <Dialog open={open} onOpenChange={handleDialogOpenChange}>
      <DialogTrigger asChild>{trigger}</DialogTrigger>
      <DialogContent className="border-border">
        <DialogHeader>
          <DialogTitle>Create a new permission</DialogTitle>
          <DialogDescription>Permissions allow your key to do certain actions.</DialogDescription>
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
                    <Input placeholder="domain.create" {...field} />
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
                  <FormLabel>
                    Description{" "}
                    <Badge variant="secondary" size="sm">
                      Optional
                    </Badge>
                  </FormLabel>
                  <FormControl>
                    <Textarea
                      rows={3}
                      placeholder="Create a new domain in this account."
                      {...field}
                      onBlur={(e) => {
                        if (e.target.value === "") {
                          return;
                        }
                      }}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button type="submit">
                {createPermission.isLoading ? (
                  <Loading className="w-4 h-4" />
                ) : (
                  "Create New Permission"
                )}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
