"use client";
import { Button } from "@/components/ui/button";
import React, { useState } from "react";

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

type Props = {
  api: {
    id: string;
    workspaceId: string;
    name: string;
  };
};

const intent = "delete my api";

export const DeleteApi: React.FC<Props> = ({ api }) => {
  const [open, setOpen] = useState(false);

  const formSchema = z.object({
    name: z.string().refine((v) => v === api.name, "Please confirm the API name"),
    intent: z.string().refine((v) => v === intent, "Please confirm your intent"),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const router = useRouter();

  const deleteApi = trpc.api.delete.useMutation({
    onSuccess() {
      toast.message("API Deleted", {
        description: "Your API and all its keys are being deleted now.",
      });

      router.replace("/app/apis");
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  const isValid = form.watch("intent") === intent && form.watch("name") === api.name;

  async function onSubmit(_values: z.infer<typeof formSchema>) {
    deleteApi.mutate({ apiId: api.id });
  }

  return (
    <>
      <Card className="relative border-2 border-[#b80f07]">
        <CardHeader>
          <CardTitle>Delete</CardTitle>
          <CardDescription>
            This api will be deleted, along with all of its keys and data. This action cannot be
            undone.
          </CardDescription>
        </CardHeader>

        <CardFooter className="z-10 justify-end">
          <Button type="button" onClick={() => setOpen(!open)} variant="alert">
            Delete API
          </Button>
        </CardFooter>
      </Card>
      <Dialog open={open} onOpenChange={(o) => setOpen(o)}>
        <DialogContent className="border-[#b80f07]">
          <DialogHeader>
            <DialogTitle>Delete API</DialogTitle>
            <DialogDescription>
              This api will be deleted, along with all of its keys. This action cannot be undone.
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
                      Enter the API name{" "}
                      <span className="font-medium text-content">{api.name}</span> to continue:
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
                      To verify, type{" "}
                      <span className="font-medium text-content">delete my api</span> below:
                    </FormLabel>
                    <FormControl>
                      <Input {...field} autoComplete="off" />
                    </FormControl>

                    <FormMessage />
                  </FormItem>
                )}
              />

              <DialogFooter className="justify-end gap-4">
                <Button type="button" onClick={() => setOpen(!open)} variant="secondary">
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant={isValid ? "alert" : "disabled"}
                  disabled={!isValid || form.formState.isLoading}
                >
                  {form.formState.isLoading ? <Loading /> : "Delete API"}
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </>
  );
};
