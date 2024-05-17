"use client";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
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
import { Switch } from "@/components/ui/switch";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import type { Key } from "@unkey/db";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  keyId: z.string(),
  workspaceId: z.string(),
  enabled: z.boolean(),
  ratelimitType: z.enum(["fast", "consistent"]).optional().default("fast"),
  ratelimitLimit: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message: issue.code === "invalid_type" ? "Amount must be greater than 0" : defaultError,
      }),
    })
    .positive({ message: "This refill limit must be a positive number." })
    .int()
    .optional(),
  ratelimitRefillRate: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message: issue.code === "invalid_type" ? "Amount must be greater than 0" : defaultError,
      }),
    })
    .positive({ message: "This refill rate must be a positive number." })
    .int()
    .optional(),
  ratelimitRefillInterval: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message: issue.code === "invalid_type" ? "Amount must be greater than 0" : defaultError,
      }),
    })
    .positive({ message: "This refill interval must be a positive number." })
    .int()
    .optional(),
});

type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    ratelimitType: Key["ratelimitType"];
    ratelimitLimit: number | null;
    ratelimitRefillRate: number | null;
    ratelimitRefillInterval: number | null;
  };
};

export const UpdateKeyRatelimit: React.FC<Props> = ({ apiKey }) => {
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      keyId: apiKey.id,
      workspaceId: apiKey.workspaceId,
      enabled: apiKey.ratelimitType !== null,
      ratelimitType: apiKey.ratelimitType ? apiKey.ratelimitType : undefined,
      ratelimitLimit: apiKey.ratelimitLimit ? apiKey.ratelimitLimit : undefined,
      ratelimitRefillRate: apiKey.ratelimitRefillRate ? apiKey.ratelimitRefillRate : undefined,
      ratelimitRefillInterval: apiKey.ratelimitRefillInterval
        ? apiKey.ratelimitRefillInterval
        : undefined,
    },
  });
  const updateRatelimit = trpc.key.update.ratelimit.useMutation({
    onSuccess() {
      toast.success("Your ratelimit has been updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });
  async function onSubmit(values: z.infer<typeof formSchema>) {
    updateRatelimit.mutateAsync(values);
  }
  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardHeader>
            <CardTitle>Ratelimit</CardTitle>
            <CardDescription>How frequently this key can be used.</CardDescription>
          </CardHeader>
          <CardContent className="flex flex-col item-center">
            <div
              className={cn("flex flex-col", {
                "opacity-50": !form.getValues("enabled"),
              })}
            >
              <div className="flex flex-col gap-1">
                <Label htmlFor="ratelimitLimit">Limit</Label>
                <FormField
                  control={form.control}
                  name="ratelimitLimit"
                  render={({ field }) => (
                    <FormItem>
                      <FormControl>
                        <Input
                          {...field}
                          disabled={!form.getValues("enabled")}
                          type="number"
                          min={0}
                          className="max-w-sm"
                          defaultValue={apiKey.ratelimitLimit ?? undefined}
                          autoComplete="off"
                        />
                      </FormControl>
                      <FormMessage />
                    </FormItem>
                  )}
                />
                <FormDescription>
                  The maximum number of requests in the given fixed window.
                </FormDescription>
              </div>
              <div className="flex flex-col gap-1">
                <Label htmlFor="ratelimitRefillRate">Refill Rate</Label>
                <FormField
                  control={form.control}
                  name="ratelimitRefillRate"
                  render={({ field }) => (
                    <FormItem>
                      <FormControl>
                        <Input
                          {...field}
                          disabled={!form.getValues("enabled")}
                          type="number"
                          min={0}
                          className="max-w-sm"
                          defaultValue={apiKey.ratelimitRefillRate ?? undefined}
                          autoComplete="off"
                        />
                      </FormControl>
                      <FormDescription>
                        <strong>Refill Rate</strong> is the number of requests that are allowed per{" "}
                        <strong>Refill Interval</strong>.
                      </FormDescription>
                    </FormItem>
                  )}
                />
              </div>
              <div className="flex flex-col gap-1">
                <Label htmlFor="ratelimitRefillInterval">
                  Refill Interval{" "}
                  <span className="text-xs text-content-subtle">(milliseconds)</span>
                </Label>
                <FormField
                  control={form.control}
                  name="ratelimitRefillInterval"
                  render={({ field }) => (
                    <FormItem>
                      <FormControl>
                        <Input
                          {...field}
                          disabled={!form.getValues("enabled")}
                          type="number"
                          min={0}
                          className="max-w-sm"
                          defaultValue={apiKey.ratelimitRefillInterval ?? undefined}
                          autoComplete="off"
                        />
                      </FormControl>
                      <FormDescription>
                        How many requests may be performed in a given interval
                      </FormDescription>
                      <FormMessage />
                    </FormItem>
                  )}
                />
              </div>
            </div>
          </CardContent>
          <CardFooter className="justify-between">
            <FormField
              control={form.control}
              name="enabled"
              render={({ field }) => (
                <FormItem className="w-full">
                  <div className="flex items-center gap-4">
                    <FormControl>
                      <Switch
                        checked={form.getValues("enabled")}
                        onCheckedChange={(e) => {
                          field.onChange(e);
                        }}
                      />
                    </FormControl>{" "}
                    <FormLabel htmlFor="enabled">
                      {form.getValues("enabled") ? "Enabled" : "Disabled"}
                    </FormLabel>
                  </div>
                </FormItem>
              )}
            />
            <Button
              disabled={updateRatelimit.isLoading || !form.formState.isValid}
              className="mt-4 "
              type="submit"
            >
              {updateRatelimit.isLoading ? <Loading /> : "Save"}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
