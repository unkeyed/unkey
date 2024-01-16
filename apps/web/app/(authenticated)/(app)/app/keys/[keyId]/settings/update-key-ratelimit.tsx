"use client";
import { SubmitButton } from "@/components/dashboard/submit-button";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { FormField } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Key } from "@unkey/db";
import { useRouter } from "next/navigation";
import React, { useState } from "react";
import { Form, useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  keyId: z.string(),
  enabled: z.boolean(),
  ratelimitType: z.string().optional(),
  ratelimitLimit: z.string().optional(),
  ratelimitRefillRate: z.string().optional(),
  ratelimitRefillInterval: z.string().optional(),
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
  const [enabled, setEnabled] = useState(apiKey.ratelimitType !== null);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const updateRatelimit = trpc.keySettings.updateRatelimit.useMutation({
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
    updateRatelimit.mutate(values);
  }
  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardHeader>
            <CardTitle>Ratelimit</CardTitle>
            <CardDescription>How frequently this key can be used.</CardDescription>
          </CardHeader>
          <CardContent className="relative flex justify-between item-center">
            <div className={cn("flex flex-col", { "opacity-50": !enabled })}>
              <input type="hidden" name="keyId" value={apiKey.id} />
              <input type="hidden" name="enabled" value={enabled ? "true" : "false"} />
              <input type="hidden" name="ratelimitType" value={enabled ? "fast" : undefined} />

              <div className="flex flex-col gap-1">
                <Label htmlFor="ratelimitLimit">Limit</Label>
                <FormField
                  control={form.control}
                  name="ratelimitLimit"
                  render={({ field }) => (
                    <Input
                      disabled={!enabled}
                      {...field}
                      type="number"
                      min={0}
                      className="max-w-sm"
                      defaultValue={apiKey.ratelimitLimit ?? undefined}
                      autoComplete="off"
                    />
                  )}
                />

                <p className="mt-1 text-xs text-content-subtle">
                  The maximum number of requests possible during a burst.
                </p>
              </div>
              <div className="flex items-center justify-between w-full gap-4 mt-8">
                <div className="flex flex-col gap-1">
                  <Label htmlFor="ratelimitRefillRate">Refill Rate</Label>
                  <FormField
                    control={form.control}
                    name="ratelimitRefillRate"
                    render={({ field }) => (
                      <Input
                        disabled={!enabled}
                        {...field}
                        type="number"
                        min={0}
                        className="max-w-sm"
                        defaultValue={apiKey.ratelimitRefillRate ?? undefined}
                        autoComplete="off"
                      />
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
                      <Input
                        disabled={!enabled}
                        type="number"
                        min={0}
                        className="max-w-sm"
                        defaultValue={apiKey.ratelimitRefillInterval ?? undefined}
                        autoComplete="off"
                      />
                    )}
                  />
                </div>
              </div>
              <p className="mt-1 text-xs text-content-subtle">
                How many requests may be performed in a given interval
              </p>
            </div>
          </CardContent>
          <CardFooter className="justify-between">
            <div className="flex items-center gap-4">
              <Switch id="enabled" checked={enabled} onCheckedChange={setEnabled} />
              <Label htmlFor="enabled">{enabled ? "Enabled" : "Disabled"}</Label>
            </div>
            <SubmitButton label="Save" />
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
