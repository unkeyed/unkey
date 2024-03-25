"use client";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardFooter, CardHeader, CardTitle } from "@/components/ui/card";
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
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import React from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  identifier: z.string().min(2).max(250),
  limit: z.coerce.number().int().min(1).max(1_000),
  duration: z.coerce
    .number()
    .int()
    .min(1_000)
    .max(24 * 60 * 60 * 1000),
});

type Props = {
  namespaceId: string;
};

export const CreateNewOverride: React.FC<Props> = ({ namespaceId }) => {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    reValidateMode: "onChange",
    defaultValues: {
      limit: 10,
      duration: 60_000,
    },
  });

  const create = trpc.ratelimit.override.create.useMutation({
    onSuccess() {
      toast.success("New override has been created", {
        description: "Changes may take up to 60s to propagate globally",
      });
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });
  async function onSubmit(values: z.infer<typeof formSchema>) {
    create.mutate({ ...values, namespaceId });
  }
  const router = useRouter();

  const error =
    form.formState.errors.duration ??
    form.formState.errors.identifier ??
    form.formState.errors.limit;

  return (
    <Card>
      <CardHeader>
        <CardTitle>New Override</CardTitle>
      </CardHeader>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <CardContent className="grid grid-cols-3 gap-4">
            <FormField
              control={form.control}
              name="identifier"
              render={({ field }) => (
                <FormItem className="col-span-1">
                  <FormLabel>Identifier</FormLabel>
                  <FormControl>
                    <Input
                      placeholder="hello@user.xyz"
                      {...field}
                      className=" dark:focus:border-gray-700"
                    />
                  </FormControl>
                  <FormDescription>The identifier you use when ratelimiting.</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="limit"
              render={({ field }) => (
                <FormItem className="col-span-1">
                  <FormLabel>Limit</FormLabel>
                  <FormControl>
                    <Input {...field} className=" dark:focus:border-gray-700" />
                  </FormControl>
                  <FormDescription>
                    How many request can be made within a given window.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
            <FormField
              control={form.control}
              name="duration"
              render={({ field }) => (
                <FormItem className="col-span-1">
                  <FormLabel>Duration</FormLabel>
                  <FormControl>
                    <Input type="number" {...field} className="dark:focus:border-gray-700" />
                  </FormControl>
                  <FormDescription>Duration of each window in milliseconds.</FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
          <CardFooter className="flex flex-row-reverse justify-between">
            <Button disabled={create.isLoading || !form.formState.isValid} type="submit">
              {create.isLoading ? <Loading /> : "Create"}
            </Button>
            {error ? <span className="text-sm text-alert">{error.message}</span> : null}
          </CardFooter>
        </form>
      </Form>
    </Card>
  );
};
