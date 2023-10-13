"use client";
import React, { useState } from "react";

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
import { updateKeyRemaining } from "./actions";
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    remainingRequests: number | null;
  };
};

export const UpdateKeyRemaining: React.FC<Props> = ({ apiKey }) => {
  const { toast } = useToast();

  const [enabled, setEnabled] = useState(apiKey.remainingRequests !== null);
  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateKeyRemaining(formData);
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
          description: "Remaining uses updated",
        });
      }}
    >
      <Card>
        <CardHeader>
          <CardTitle>Remaining Uses</CardTitle>
          <CardDescription>
            How many times this key can be used before it gets disabled automatically.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex justify-between item-center">
          <div className={cn("flex flex-col space-y-2", { "opacity-50": !enabled })}>
            <input type="hidden" name="keyId" value={apiKey.id} />
            <input type="hidden" name="enableRemaining" value={enabled ? "true" : "false"} />

            <Label htmlFor="remaining">Remaining</Label>
            <Input
              disabled={!enabled}
              type="number"
              min={0}
              name="remaining"
              className="max-w-sm"
              defaultValue={apiKey.remainingRequests ?? ""}
              autoComplete="off"
            />
          </div>
        </CardContent>
        <CardFooter className="justify-between">
          <div className="flex items-center gap-4">
            <Switch id="enableRemaining" checked={enabled} onCheckedChange={setEnabled} />
            <Label htmlFor="enableRemaining">{enabled ? "Enabled" : "Disabled"}</Label>
          </div>
          <SubmitButton label="Save" />
        </CardFooter>
      </Card>
    </form>
  );
};
