"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Loading } from "@/components/dashboard/loading";
import { VisibleButton } from "@/components/dashboard/visible-button";
import { Button } from "@/components/ui/button";
import { Code } from "@/components/ui/code";

import { MultiSelect } from "@/components/multi-select";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
import { Label } from "@/components/ui/label";
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
  name: z.string(),
  key: z.string(),
  description: z.string().optional(),
});

export const CreateNewRole: React.FC<Props> = ({ trigger }) => {
  const router = useRouter();

  const [createMore, setCreateMore] = useState(false);
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const createRole = trpc.rbac.createRole.useMutation({
    onSuccess({ roleId }) {
      const href = `/app/settings/roles/${roleId}`;
      router.prefetch(href);
      toast.success("your role was created", {
        action: createMore
          ? {
              label: "Go to role",
              onClick: () => router.push(href),
            }
          : undefined,
      });
      if (!createMore) {
        return router.push(href);
      }
      form.reset({
        name: "",
        key: "",
      });
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
    <Dialog>
      <DialogTrigger>{trigger}</DialogTrigger>
      <DialogContent>
        <DialogHeader>
          <DialogTitle>Create a new role</DialogTitle>
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
                  <FormDescription>A human readable name for your role</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="key"
              render={({ field }) => (
                <FormItem>
                  <FormLabel>Key</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="org.domains.manage"
                      {...field}
                      className=" dark:focus:border-gray-700"
                    />
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
              <div className="flex items-center gap-1">
                <Checkbox
                  id="create-more"
                  checked={createMore}
                  onClick={() => setCreateMore(!createMore)}
                />
                <Label htmlFor="create-more" className="text-xs">
                  Create more
                </Label>
              </div>

              <Button type="submit">
                {createRole.isLoading ? <Loading className="w-4 h-4" /> : "Create New Role"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
