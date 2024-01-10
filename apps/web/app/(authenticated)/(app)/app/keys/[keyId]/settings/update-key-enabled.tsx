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
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc/client";
import router from "next/router";
import React, { useState } from "react";
import { updateKeyEnabled } from "./actions";
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    enabled: boolean;
  };
};

export const UpdateKeyEnabled: React.FC<Props> = ({ apiKey }) => {
  const { toast } = useToast();
  const router = useRouter();
  const [enabled, setEnabled] = useState(apiKey.enabled);
  const updateEnabled = trpc.keySettings.updateName.useMutation({
    onSuccess: (_data) => {
      toast({
        title: "Success",
        description: "Your Api name has been updated!",
      });
      router.refresh();
    },
    onError: (err, variables) => {
      router.refresh();
      toast({
        title: `Could not update Api name on ApiId ${variables.apiId}`,
        description: err.message,
        variant: "alert",
      });
    },
  });
  function handleSubmit(event: any) {
    event.preventDefault();
    const formData = new FormData(event.target);
    const apiName = formData.get("name");
    const apiId = formData.get("apiId");
    const workspaceId = formData.get("workspaceId");

    updateEnabled.mutate({
      name: apiName as string,
      apiId: apiId as string,
      workspaceId: workspaceId as string,
    });
  }
  return (
    <form onSubmit={handleSubmit}>
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
