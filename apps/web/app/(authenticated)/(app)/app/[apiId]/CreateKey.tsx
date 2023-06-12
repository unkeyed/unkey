"use client";

import { useToast } from "@/components/ui/use-toast";
import { useRouter } from "next/navigation";

import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/Form";
import { Loading } from "@/components/loading";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogContent,
  DialogDescription,
  DialogFooter,
  DialogHeader,
  DialogTitle,
  DialogTrigger,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useState } from "react";
import { CopyButton } from "@/components/CopyButton";

const formSchema = z.object({
  bytes: z.number().int().gte(1),
  prefix: z.string().max(8).optional(),
  ownerId: z.string().optional(),
  meta: z.record(z.unknown()).optional(),
});
type Props = {
  apiId: string;
};

export const CreateKeyButton: React.FC<Props> = ({ apiId }) => {
  const { toast } = useToast();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    defaultValues: {
      prefix: "api",
      bytes: 16,
    },
  });
  const router = useRouter();
  const key = trpc.key.create.useMutation({
    onSuccess() {
      toast({
        title: "Key Created",
        description: "Your Key has been created",
      });
    },
    onError(err) {
      console.error(err);
      toast({ title: "Error", description: err.message, variant: "destructive" });
    },
  });
  async function onSubmit(values: z.infer<typeof formSchema>) {
    await key.mutateAsync({
      apiId,
      ...values,
      ownerId: values.ownerId ?? undefined,
    });
  }

  const snippet = `curl 'https://unkey.dev/api/v1/TODO' \\
  -H 'Authorization: Bearer ${key.data?.key}' \\
  -H 'Content-Type: application/json' \\
  -d '{
    "TODO": true
  }'
  `;

  return (
    <>
      <Dialog
        onOpenChange={(v) => {
          if (!v) {
            // Remove the key from memory when closing the modal
            key.reset();
            router.refresh();
          }
        }}
      >
        <DialogTrigger asChild>
          <Button>Create Key</Button>
        </DialogTrigger>

        {key.data ? (
          <DialogContent>
            <DialogHeader>
              <DialogTitle>Your API Key</DialogTitle>
              <DialogDescription>
                This key is only shown once and can not be recovered. Please store it somewhere
                safe.
              </DialogDescription>

              <div className="flex items-center justify-between px-2 py-1 mt-4 border rounded  gap-4 lg:p-4 border-white/10 bg-zinc-100 dark:bg-zinc-900">
                <pre className="font-mono">{key.data.key}</pre>
                <CopyButton value={key.data.key} />
              </div>
            </DialogHeader>

            <p className="mt-2 text-sm font-medium text-center text-zinc-100 ">
              Try it out with curl
            </p>
            <div className="flex items-start justify-between px-2 py-1 border rounded gap-4 lg:p-4 border-white/10  bg-zinc-100 dark:bg-zinc-900">
              <pre className="font-mono">{snippet}</pre>
              <CopyButton value={snippet} />
            </div>
          </DialogContent>
        ) : (
          <DialogContent>
            <DialogTitle>Create a new API</DialogTitle>

            <Form {...form}>
              <form onSubmit={form.handleSubmit(onSubmit)} className="flex flex-col space-y-4">
                <FormField
                  control={form.control}
                  name="prefix"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Prefix</FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                      <FormDescription>
                        Using a prefix can make it easier for your users to distinguis between api
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="bytes"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Bytes</FormLabel>
                      <FormControl>
                        <Input type="number" {...field} />
                      </FormControl>
                      <FormDescription>How many bytes to use.</FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormField
                  control={form.control}
                  name="ownerId"
                  render={({ field }) => (
                    <FormItem>
                      <FormLabel>Owner</FormLabel>
                      <FormControl>
                        <Input {...field} />
                      </FormControl>
                      <FormDescription>
                        This is the id of the user or tenant in your system, so you can identify
                        users from an API key.
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />

                <DialogFooter className="justify-end">
                  <Button type="submit">{key.isLoading ? <Loading /> : "Create"}</Button>
                </DialogFooter>
              </form>
            </Form>
          </DialogContent>
        )}
      </Dialog>
    </>
  );
};
