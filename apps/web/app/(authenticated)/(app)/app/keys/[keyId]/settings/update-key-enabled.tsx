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
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { toast } from "@/components/ui/toaster";
import { updateKeyEnabled } from "./actions";
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    enabled: boolean;
  };
};

export const UpdateKeyEnabled: React.FC<Props> = ({ apiKey }) => {
  const [enabled, setEnabled] = useState(apiKey.enabled);

  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateKeyEnabled(formData);
        if (res.error) {
          toast.error(res.error.message);
          return;
        }
        toast.success("Enabled has been updated");
      }}
    >
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
            <input type="hidden" name="enabled" value={enabled ? "true" : "false"} />
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
          <SubmitButton label="Save" />
        </CardFooter>
      </Card>
    </form>
  );
};
