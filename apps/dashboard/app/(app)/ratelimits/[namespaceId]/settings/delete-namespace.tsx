"use client";
import { Button } from "@unkey/ui";
import type React from "react";
import { useState } from "react";

import { Card, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";

import { Loading } from "@/components/dashboard/loading";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
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
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { revalidate } from "./actions";

type Props = {
  namespace: {
    id: string;
    workspaceId: string;
    name: string;
  };
};

const intent = "delete namespace";

export const DeleteNamespace: React.FC<Props> = ({ namespace }) => {
  const [open, setOpen] = useState(false);

  const formSchema = z.object({
    name: z.string().refine((v) => v === namespace.name, "Please confirm the namespace name"),
    intent: z.string().refine((v) => v === intent, "Please confirm your intent"),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const router = useRouter();

  const deleteNamespace = trpc.ratelimit.namespace.delete.useMutation({
    async onSuccess() {
      toast.message("Namespace Deleted", {
        description: "Your namespace and all its overridden identifiers have been deleted.",
      });

      await revalidate();

      router.push("/ratelimits");
    },
    onError(err) {
      toast.error(err.message);
    },
  });

  const isValid = form.watch("intent") === intent && form.watch("name") === namespace.name;

  async function onSubmit(_values: z.infer<typeof formSchema>) {
    deleteNamespace.mutate({ namespaceId: namespace.id });
  }

  function handleDialogOpenChange(newState: boolean) {
    setOpen(newState);
    form.reset();
  }

  return (
    <>
      <Card className="relative border-2 border-alert">
        <CardHeader>
          <CardTitle>Delete</CardTitle>
          <CardDescription>
            This namespace will be deleted, along with all of its identifiers and data. This action
            cannot be undone.
          </CardDescription>
        </CardHeader>

        <CardFooter className="z-10 justify-end">
          <Button type="button" onClick={() => setOpen(!open)} variant="destructive">
            Delete namespace
          </Button>
        </CardFooter>
      </Card>
      <Dialog open={open} onOpenChange={handleDialogOpenChange}>
        <DialogContent className="border-alert">
          <DialogHeader>
            <DialogTitle>Delete namespace</DialogTitle>
            <DialogDescription>
              This namespace will be deleted, along with all of its identifiers and data. This
              action cannot be undone.
            </DialogDescription>
          </DialogHeader>
          <Form {...form}>
            <form className="flex flex-col space-y-8" onSubmit={form.handleSubmit(onSubmit)}>
              <Alert variant="alert">
                <AlertTitle>Warning</AlertTitle>
                <AlertDescription>
                  This action is not reversible. Please be certain.
                </AlertDescription>
              </Alert>

              <FormField
                control={form.control}
                name="name"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className="font-normal text-content-subtle">
                      {" "}
                      Enter the namespace name{" "}
                      <span className="font-medium text-content">{namespace.name}</span> to
                      continue:
                    </FormLabel>
                    <FormControl>
                      <Input {...field} autoComplete="off" />
                    </FormControl>

                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="intent"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel className="font-normal text-content-subtle">
                      To verify, type <span className="font-medium text-content">{intent}</span>{" "}
                      below:
                    </FormLabel>
                    <FormControl>
                      <Input {...field} autoComplete="off" />
                    </FormControl>

                    <FormMessage />
                  </FormItem>
                )}
              />

              <DialogFooter className="justify-end gap-4">
                <Button
                  type="button"
                  disabled={deleteNamespace.isLoading}
                  onClick={() => setOpen(!open)}
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant="destructive"
                  disabled={!isValid || deleteNamespace.isLoading}
                >
                  {deleteNamespace.isLoading ? <Loading /> : "Delete namespace"}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </>
  );
};
