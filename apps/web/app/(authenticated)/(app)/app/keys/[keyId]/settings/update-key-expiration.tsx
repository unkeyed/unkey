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
import { cn } from "@/lib/utils";
import { updateExpiration } from "./actions";
import format from "date-fns/format";
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

  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateExpiration(formData);
        if (res.error) {
          toast({
            title: "Error",
            description: res.error.message,
            variant: "alert",
          });
          return;
        }
        toast({
          title: "Success",
          description: "Expiration updated",
        });
      }}
    >
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
            <Input
              disabled={!enabled}
              type="datetime-local"
              name="expiration"
              className="max-w-sm"
              defaultValue={
                apiKey.expires ? format(new Date(apiKey.expires), "yyyy-MM-dd'T'HH:mm") : ""
              }
              placeholder={placeholder}
              autoComplete="off"
            />
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
