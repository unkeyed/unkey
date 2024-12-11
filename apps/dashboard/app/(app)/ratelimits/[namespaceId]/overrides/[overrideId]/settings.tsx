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
import { useRouter } from "next/navigation";
import type React from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  limit: z.coerce.number().int().min(1).max(10_000),
  duration: z.coerce
    .number()
    .int()
    .min(1_000)
    .max(24 * 60 * 60 * 1000),
  async: z.enum(["unset", "sync", "async"]),
});

type Props = {
  overrideId: string;
  defaultValues: {
    limit: number;
    duration: number;
    async?: boolean;
  };
};

export const UpdateCard: React.FC<Props> = ({ overrideId, defaultValues }) => {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    reValidateMode: "onChange",
    defaultValues: {
      limit: defaultValues.limit,
      duration: defaultValues.duration,
      async:
        typeof defaultValues.async === "undefined"
          ? "unset"
          : defaultValues.async
            ? "async"
            : "sync",
    },
  });

  const update = trpc.ratelimit.override.update.useMutation({
    onSuccess() {
      toast.success("Limits have been updated", {
        description: "Changes may take up to 60s to propagate globally",
      });
      router.refresh();
    },
    onError(err) {
      toast.error(err.message);
    },
  });

  const deleteOverride = trpc.ratelimit.override.delete.useMutation({
    onSuccess() {
      toast.success("Override has been deleted", {
        description: "Changes may take up to 60s to propagate globally",
      });
      router.push("/ratelimits/");
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    update.mutate({
      id: overrideId,
      limit: values.limit,
      duration: values.duration,
      async: {
        unset: null,
        sync: false,
        async: true,
      }[values.async],
    });
  }
  const router = useRouter();

  return (
    <Card>
      <CardHeader>
        <CardTitle>Limit</CardTitle>
      </CardHeader>

      <Form {...form}>
        <form onSubmit={form.handleSubmit(onSubmit)}>
          <CardContent className="grid grid-cols-5 gap-4">
            <FormField
              control={form.control}
              name="limit"
              render={({ field }) => (
                <FormItem className="col-span-2">
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
                <FormItem className="col-span-2">
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
                  <Select onValueChange={field.onChange} value={field.value}>
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

          <CardFooter className="flex justify-end space-x-4">
            <Button type="button" onClick={() => deleteOverride.mutate({ id: overrideId })}>
              Delete
            </Button>
            <Button disabled={update.isLoading || !form.formState.isValid} type="submit">
              {update.isLoading ? <Loading /> : "Save"}
            </Button>
          </CardFooter>
        </form>
      </Form>
    </Card>
  );
};
