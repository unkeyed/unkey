"use client";
import { Button } from "@/components/ui/button";
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
import { PostHogEvent } from "@/providers/PostHogProvider";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { revalidate } from "./actions";

type Props = {
  gateway: {
    id: string;
    name: string;
  };
};

const intent = "delete gateway";

export const DeleteGateway: React.FC<Props> = ({ gateway }) => {
  const [open, setOpen] = useState(false);

  const formSchema = z.object({
    name: z.string().refine((v) => v === gateway.name, "Please confirm the gateway name"),
    intent: z.string().refine((v) => v === intent, "Please confirm your intent"),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const router = useRouter();

  const deleteGateway = trpc.llmGateway.delete.useMutation({
    async onSuccess(res) {
      toast.message("Gateway Deleted", {
        description: "Your gateway has been deleted.",
      });

      PostHogEvent({
        name: "semantic_cache_gateway_deleted",
        properties: { id: res.id },
      });

      await revalidate();

      router.push("/");
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  const isValid = form.watch("intent") === intent && form.watch("name") === gateway.name;

  async function onSubmit(_values: z.infer<typeof formSchema>) {
    deleteGateway.mutate({ gatewayId: gateway.id });
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
            This gateway will be deleted. This action cannot be undone.
          </CardDescription>
        </CardHeader>

        <CardFooter className="z-10 justify-start sm:justify-end">
          <Button type="button" onClick={() => setOpen(!open)} variant="alert">
            Delete gateway
          </Button>
        </CardFooter>
      </Card>
      <Dialog open={open} onOpenChange={handleDialogOpenChange}>
        <DialogContent className="border-alert">
          <DialogHeader>
            <DialogTitle>Delete gateway</DialogTitle>
            <DialogDescription>
              This gateway will be deleted. This action cannot be undone.
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
                      Enter the gateway name{" "}
                      <span className="font-medium text-content">{gateway.name}</span> to continue:
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
                  disabled={deleteGateway.isLoading}
                  onClick={() => setOpen(!open)}
                  variant="secondary"
                >
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant={isValid ? "alert" : "disabled"}
                  disabled={!isValid || deleteGateway.isLoading}
                >
                  {deleteGateway.isLoading ? <Loading /> : "Delete gateway"}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </>
  );
};
