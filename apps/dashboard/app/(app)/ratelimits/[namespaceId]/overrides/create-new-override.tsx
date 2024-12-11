"use client";
import { Loading } from "@/components/dashboard/loading";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { useRouter, useSearchParams } from "next/navigation";
import type React from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  identifier: z
    .string()
    .trim()
    .min(3, "Name is required and should be at least 3 characters")
    .max(250),
  limit: z.coerce.number().int().min(1).max(10_000),
  duration: z.coerce
    .number()
    .int()
    .min(1_000)
    .max(24 * 60 * 60 * 1000),
  async: z.enum(["unset", "sync", "async"]),
});

type Props = {
  namespaceId: string;
};

export const CreateNewOverride: React.FC<Props> = ({ namespaceId }) => {
  const searchParams = useSearchParams();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    reValidateMode: "onChange",
    defaultValues: {
      limit: 10,
      duration: 60_000,
      async: "unset",
      identifier: searchParams?.get("identifier") ?? undefined,
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
      toast.error(err.message);
    },
  });
  async function onSubmit(values: z.infer<typeof formSchema>) {
    create.mutate({
      namespaceId,
      identifier: values.identifier,
      limit: values.limit,
      duration: values.duration,
      async: {
        unset: undefined,
        sync: false,
        async: true,
      }[values.async],
    });
  }
  const router = useRouter();

  return (
    <Card>
      <CardHeader>
        <CardTitle>New Override</CardTitle>
      </CardHeader>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <CardContent className="grid grid-cols-4 gap-4">
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
            <FormField
              control={form.control}
              name="async"
              render={({ field }) => (
                <FormItem className="col-span-1">
                  <FormLabel>Async</FormLabel>
                  <Select onValueChange={field.onChange} defaultValue="unset" value={field.value}>
                    <SelectTrigger>
                      <SelectValue />
                    </SelectTrigger>
                    <SelectContent>
                      <SelectItem value="unset">Don't override</SelectItem>
                      <SelectItem value="async">Async</SelectItem>
                      <SelectItem value="sync">Sync</SelectItem>
                    </SelectContent>
                  </Select>
                  <FormDescription>
                    Override the mode, async is faster but slightly less accurate.
                  </FormDescription>
                  <FormMessage />
                </FormItem>
              )}
            />
          </CardContent>
          <CardFooter className="flex flex-row-reverse justify-between">
            <Button disabled={create.isLoading || !form.formState.isValid} type="submit">
              {create.isLoading ? <Loading /> : "Create"}
            </Button>
          </CardFooter>
        </form>
      </Form>
    </Card>
  );
};
