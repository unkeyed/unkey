"use client";

import { Loading } from "@/components/dashboard/loading";
import { MultiSelect } from "@/components/multi-select";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";

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
import type { Permission } from "@unkey/db";
import { useRouter } from "next/navigation";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

type Props = {
  trigger: React.ReactNode;
  permissions?: Permission[];
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
  permissionOptions: z
    .array(
      z.object({
        label: z.string(),
        value: z.string(),
      }),
    )
    .optional(),
});

export const CreateNewRole: React.FC<Props> = ({ trigger, permissions }) => {
  const router = useRouter();
  const [open, setOpen] = useState(false);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    reValidateMode: "onBlur",
  });

  const createRole = trpc.rbac.createRole.useMutation({
    onSuccess({ roleId }) {
      toast.success("Role created");

      form.reset({
        name: "",
        description: "",
      });
      setOpen(false);
      router.push(`/app/authorization/roles/${roleId}`);
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    createRole.mutate({
      name: values.name,
      description: values.description,
      permissionIds: values.permissionOptions?.map((o) => o.value),
    });
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
                    <Input placeholder="domain.manager" {...field} />
                  </FormControl>
                  <FormDescription>
                    A unique name for your role. You will use this when managing roles through the
                    API. These are not customer facing.
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
                      placeholder="Manage domains and DNS records "
                      {...field}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            {permissions && permissions.length > 0 ? (
              <FormField
                control={form.control}
                name="permissionOptions"
                render={({ field }) => (
                  <FormItem>
                    {" "}
                    <FormLabel>
                      Add existing permissions{" "}
                      <Badge variant="secondary" size="sm">
                        Optional
                      </Badge>
                    </FormLabel>
                    <MultiSelect
                      options={permissions.map((p) => ({ label: p.name, value: p.id }))}
                      selected={field.value ?? []}
                      setSelected={(cb) => {
                        if (typeof cb === "function") {
                          return form.setValue("permissionOptions", cb(field.value ?? []));
                        }
                      }}
                    />
                    <FormMessage />
                  </FormItem>
                )}
              />
            ) : null}
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
