"use client";
import { Button } from "@/components/ui/button";
import type React from "react";
import { useState } from "react";

import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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

import { DialogTrigger } from "@radix-ui/react-dialog";
import { Err } from "@unkey/error";
import { revalidate } from "../../../apis/[apiId]/settings/actions";

type Props = {
  webhook: {
    destination: string;
    id: string;
  };
};

const intent = "delete my webhook";

export const DeleteWebhook: React.FC<Props> = ({ webhook }) => {
  const [open, setOpen] = useState(false);

  const formSchema = z.object({
    intent: z.string().refine((v) => v === intent, "Please confirm your intent"),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const router = useRouter();

  const deleteWebhook = trpc.webhook.delete.useMutation({
    async onSuccess() {
      await revalidate();

      router.push("/settings/webhooks");
    },
  });

  const isValid = form.watch("intent") === intent;

  async function onSubmit(_values: z.infer<typeof formSchema>) {
    toast.promise(deleteWebhook.mutateAsync({ webhookId: webhook.id }), {
      loading: "Deleting your webhook...",
      success: "Your webhook has been deleted",
      error: (err) => (err instanceof Error && err.message) || "Something went wrong",
    });
  }

  return (
    <Dialog open={open} onOpenChange={(o) => setOpen(o)}>
      <DialogTrigger>
        <Button variant="alert">Delete</Button>
      </DialogTrigger>
      <DialogContent className="border-alert">
        <DialogHeader>
          <DialogTitle>Delete webhook</DialogTitle>
          <DialogDescription>
            This webhook will be deleted, along with all of its events. This action cannot be
            undone.
          </DialogDescription>
        </DialogHeader>
        <Form {...form}>
          <form className="flex flex-col space-y-8" onSubmit={form.handleSubmit(onSubmit)}>
            <Alert variant="alert">
              <AlertTitle>Warning</AlertTitle>
              <AlertDescription>This action is not reversible. Please be certain.</AlertDescription>
            </Alert>

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
                disabled={deleteWebhook.isLoading}
                onClick={() => setOpen(!open)}
                variant="secondary"
              >
                Cancel
              </Button>
              <Button
                type="submit"
                variant={isValid ? "alert" : "disabled"}
                disabled={!isValid || deleteWebhook.isLoading}
              >
                {deleteWebhook.isLoading ? <Loading /> : "Delete webhook"}
              </Button>
            </DialogFooter>
          </form>
        </Form>
      </DialogContent>
    </Dialog>
  );
};
