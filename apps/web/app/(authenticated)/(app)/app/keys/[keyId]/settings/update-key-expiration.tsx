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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc";
import { cn } from "@/lib/utils";

type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    expires: Date | null;
  };
};

export const UpdateKeyExpiration: React.FC<Props> = ({ apiKey }) => {
  const { toast } = useToast();

  const [enabled, setEnabled] = useState(apiKey.expires !== null);

  const placeholder = useMemo(() => {
    const t = new Date();
    t.setUTCDate(t.getUTCDate() + 7);
    t.setUTCMinutes(0, 0, 0);
    return t.toISOString();
  }, []);

  function handleSubmit(event: any) {
    event.preventDefault();
    const formData = new FormData(event.target);
    const keyId = event.target.keyId.value;

    const _updateExpiration = trpc.keySettings.updateExpiration
      .mutate({
        keyId: keyId as string,
        enableExpiration: enabled as boolean,
        expiration: formData.get("expiration") as string,
      })
      .then((response) => {
        if (response) {
          toast({
            title: "Success",
            description: "Your remaining uses has been updated!",
          });
        } else {
          toast({
            title: "Error",
            description: "Something went wrong. Please try again later",
          });
        }
      });
  }

  return (
    <form onSubmit={handleSubmit}>
      <Card>
        <CardHeader>
          <CardTitle>Expiration</CardTitle>
          <CardDescription>Automatically revoke this key after a certain date.</CardDescription>
        </CardHeader>
        <CardContent className="flex justify-between item-center">
          <div className={cn("flex flex-col gap-2 w-full", { "opacity-50": !enabled })}>
            <input type="hidden" name="keyId" value={apiKey.id} />
            <input type="hidden" name="enableExpiration" value={enabled ? "true" : "false"} />

            <Label htmlFor="expiration">Expiration</Label>
            <Input
              disabled={!enabled}
              type="string"
              name="expiration"
              className="max-w-sm"
              defaultValue={apiKey.expires?.toISOString()}
              placeholder={placeholder}
              autoComplete="off"
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
  );
};
