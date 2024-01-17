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
import { FormField } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";

import React, { useState } from "react";
import { Form, useForm } from "react-hook-form";
import { z } from "zod";
const formSchema = z.object({
  keyId: z.string(),
  workspaceId: z.string(),
  enabled: z.boolean(),
});
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    enabled: boolean;
  };
};

export const UpdateKeyEnabled: React.FC<Props> = ({ apiKey }) => {
  const router = useRouter();
  const [enabled, setEnabled] = useState(apiKey.enabled);
  const [isLoading, _setIsLoading] = useState(false);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });
  const updateEnabled = trpc.keySettings.updateEnabled.useMutation({
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
    updateEnabled.mutate(values);
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardHeader>
            <CardTitle>Enable Key</CardTitle>
            <CardDescription>
              Enable or disable this key. Disabled keys will not verify.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex justify-between item-center">
            <div className="flex flex-col space-y-2">
              <input type="hidden" name="keyId" value={apiKey.id} />
              {/*  */}
              <FormField
                control={form.control}
                name="enabled"
                render={({ field }) => (
                  <input {...field} type="hidden" value={enabled ? "true" : "false"} />
                )}
              />
              <Switch
                id="enableSwitch"
                checked={enabled}
                onCheckedChange={setEnabled}
                defaultChecked={apiKey.enabled}
              />
              <Label htmlFor="enable">{enabled ? "Enabled" : "Disabled"}</Label>
            </div>
          </CardContent>
          <CardFooter className="justify-end">
            <Button disabled={isLoading || !form.formState.isValid} className="mt-4 " type="submit">
              {isLoading ? <Loading /> : "Save"}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
