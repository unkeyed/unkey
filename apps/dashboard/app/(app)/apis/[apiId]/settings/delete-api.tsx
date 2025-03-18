"use client";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Card, CardDescription, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
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
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { formatNumber } from "@/lib/fmt";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import type React from "react";
import { useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { revalidate } from "./actions";

type Props = {
  keys: number;
  api: {
    id: string;
    workspaceId: string;
    name: string;
    deleteProtection: boolean | null;
  };
};

export const DeleteApi: React.FC<Props> = ({ api, keys }) => {
  const [open, setOpen] = useState(false);

  const intent =
    keys > 0 ? `delete this api and ${keys} key${keys > 1 ? "s" : ""}` : "delete this api";

  const formSchema = z.object({
    name: z.string().refine((v) => v === api.name, "Please confirm the API name"),
    intent: z.string().refine((v) => v === intent, "Please confirm your intent"),
  });

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const router = useRouter();

  const deleteApi = trpc.api.delete.useMutation({
    async onSuccess() {
      toast.message("API Deleted", {
        description: `Your API and ${formatNumber(keys)} keys have been deleted.`,
      });

      await revalidate();

      router.push("/apis");
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

  function handleDialogOpenChange(newState: boolean) {
    setOpen(newState);
    form.reset();
  }

  return (
    <>
      <Card
        className={cn("relative ", {
          "borrder-opacity-50": api.deleteProtection,
          "border-2 border-alert": !api.deleteProtection,
        })}
      >
        <CardHeader>
          <CardTitle>Delete</CardTitle>
          <CardDescription>
            This api will be deleted, along with all of its keys and data. This action cannot be
            undone.
          </CardDescription>
        </CardHeader>

        <CardFooter className="z-10 justify-between flex-row-reverse w-full">
          <Button
            type="button"
            disabled={!!api.deleteProtection}
            onClick={() => setOpen(!open)}
            variant="destructive"
          >
            Delete API
          </Button>
          {api.deleteProtection ? (
            <p className="text-sm text-content">
              Deletion protection is enabled, you need to disable it before deleting this API.
            </p>
          ) : null}
        </CardFooter>
      </Card>
      <Dialog open={open} onOpenChange={handleDialogOpenChange}>
        <DialogContent className="border-alert">
          <DialogHeader>
            <DialogTitle>Delete API</DialogTitle>
            <DialogDescription>
              This API will be deleted, along with ${formatNumber(keys)} keys. This action cannot be
              undone.
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
                <Button type="button" disabled={deleteApi.isLoading} onClick={() => setOpen(!open)}>
                  Cancel
                </Button>
                <Button
                  type="submit"
                  variant="destructive"
                  loading={deleteApi.isLoading}
                  disabled={!isValid || deleteApi.isLoading}
                >
                  Delete API
                </Button>
              </DialogFooter>
            </form>
          </Form>
        </DialogContent>
      </Dialog>
    </>
  );
};
