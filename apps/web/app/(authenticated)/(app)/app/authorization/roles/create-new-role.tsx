"use client";

import { Loading } from "@/components/dashboard/loading";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

import { Checkbox } from "@/components/ui/checkbox";
import {
  Dialog,
  DialogClose,
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
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

type Props = {
  trigger: React.ReactNode;
};

const formSchema = z.object({
  name: z
    .string()
    .min(3)
    .regex(/^[a-zA-Z0-9_\-\.\*]+$/, {
      message:
        "Must be at least 3 characters long and only contain alphanumeric, periods, dashes and underscores",
    }),

  description: z.string().optional(),
});

export const CreateNewRole: React.FC<Props> = ({ trigger }) => {
  const router = useRouter();
  const [open, setOpen] = useState(false);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    reValidateMode: "onBlur",
  });

  const createRole = trpc.rbac.createRole.useMutation({
    onSuccess() {
      toast.success("Role created");

      router.refresh();
      form.reset({
        name: "",
        description: "",
      });
      setOpen(false);
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    createRole.mutate(values);
  }

  return (
    <Dialog open={open} onOpenChange={setOpen}>
      <DialogTrigger>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create a new role</DialogTitle>
          <DialogDescription>Roles group permissions together.</DialogDescription>
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
                    A unique key to identify your role. We suggest using <code>.</code> (dot)
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
                      rows={form.getValues().description?.split("\n").length ?? 3}
                      placeholder="Create a new domain in this account."
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <DialogFooter>
              <Button type="submit">
                {createRole.isLoading ? <Loading className="w-4 h-4" /> : "Create"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
