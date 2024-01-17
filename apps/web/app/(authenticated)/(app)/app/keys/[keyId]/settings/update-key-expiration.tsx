"use client";
import React, { useMemo, useState } from "react";

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
import { useRouter } from "next/navigation";
import { Form, useForm } from "react-hook-form";
import { z } from "zod";
const formSchema = z.object({
  keyId: z.string(),
  enableExpiration: z.boolean(),
  expiration: z.string().optional(),
});
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    expires: Date | null;
  };
};

export const UpdateKeyExpiration: React.FC<Props> = ({ apiKey }) => {
  const [enabled, setEnabled] = useState(apiKey.expires !== null);
  const [_isLoading, _setIsLoading] = useState(false);
  const router = useRouter();
  const placeholder = useMemo(() => {
    const t = new Date();
    t.setUTCDate(t.getUTCDate() + 7);
    t.setUTCMinutes(0, 0, 0);
    return t.toISOString();
  }, []);
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const changeExpiration = trpc.keySettings.updateExpiration.useMutation({
    onSuccess() {
      toast.success("Your key has been updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    changeExpiration.mutate(values);
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardHeader>
            <CardTitle>Expiration</CardTitle>
            <CardDescription>Automatically revoke this key after a certain date.</CardDescription>
          </CardHeader>
          <CardContent className="flex justify-between item-center">
            <div
              className={cn("flex flex-col gap-2 w-full", {
                "opacity-50": !enabled,
              })}
            >
              <input type="hidden" name="keyId" value={apiKey.id} />
              <input type="hidden" name="enableExpiration" value={enabled ? "true" : "false"} />

              <Label htmlFor="expiration">Expiration</Label>
              <FormField
                control={form.control}
                name="expiration"
                render={({ field }) => (
                  <Input
                    disabled={!enabled}
                    type="string"
                    {...field}
                    className="max-w-sm"
                    defaultValue={apiKey.expires?.toISOString()}
                    placeholder={placeholder}
                    autoComplete="off"
                  />
                )}
              />

              <span className="text-xs text-content-subtle">Use ISO format: {placeholder}</span>
            </div>
          </CardContent>
          <CardFooter className="justify-between">
            <div className="flex items-center gap-4">
              <Switch id="enableExpiration" checked={enabled} onCheckedChange={setEnabled} />
              <Label htmlFor="enableExpiration">{enabled ? "Enabled" : "Disabled"}</Label>
            </div>
            <SubmitButton label="Save" />
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
