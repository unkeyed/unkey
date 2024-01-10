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
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc";
import { cn } from "@/lib/utils";
import React, { useState } from "react";
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    remaining: number | null;
    refillInterval: "daily" | "monthly" | "null";
    refillAmount: number | null;
  };
};

export const UpdateKeyRemaining: React.FC<Props> = ({ apiKey }) => {
  const { toast } = useToast();
  const [enabled, setEnabled] = useState(apiKey.remaining !== null);
  const [refillEbabled, setRefillEnabled] = useState(apiKey.remaining !== null);
  function handleSubmit(event: any) {
    event.preventDefault();
    const formData = new FormData(event.target);
    const keyId = event.target.keyId.value;

    const enableRemaining = formData.get("enableRemaining") === "true" ? true : false;
    const remaining = formData.get("remaining");
    const refillInterval = formData.get("refillInterval");
    const refillAmount = formData.get("refillAmount");
    const _updateRemaining = trpc.keySettings.updateRemaining
      .mutate({
        keyId: keyId as string,
        enableRemaining: enableRemaining as boolean,
        remaining: remaining === null ? undefined : Number(remaining),
        refillInterval: refillInterval as "daily" | "monthly" | "null" | undefined,
        refillAmount: refillInterval !== null ? Number(refillAmount) : undefined,
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
          <CardTitle>Remaining Uses</CardTitle>
          <CardDescription>
            How many times this key can be used before it gets disabled automatically.
          </CardDescription>
        </CardHeader>
        <CardContent className="flex justify-between item-center">
          <div
            className={cn("flex flex-col space-y-2", {
              "opacity-50": !enabled,
            })}
          >
            <input type="hidden" name="keyId" value={apiKey.id} />
            <input type="hidden" name="enableRemaining" value={enabled ? "true" : "false"} />

            <Label htmlFor="remaining">Remaining</Label>
            <Input
              disabled={!enabled}
              type="number"
              min={0}
              name="remaining"
              className="max-w-sm"
              defaultValue={apiKey?.remaining ?? ""}
              autoComplete="off"
            />
            <Label htmlFor="refillInterval" className="pt-4">
              Refill Rate
            </Label>
            <Select
              disabled={!enabled}
              name="refillInterval"
              defaultValue={apiKey?.refillInterval ?? "null"}
              onValueChange={(value) => setRefillEnabled(value !== "null")}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="null">None</SelectItem>
                <SelectItem value="daily">Daily</SelectItem>
                <SelectItem value="monthly">Monthly</SelectItem>
              </SelectContent>
            </Select>
            <Input
              disabled={!refillEbabled}
              name="refillAmount"
              placeholder="100"
              className="w-full"
              min={1}
              type="number"
              defaultValue={apiKey?.refillAmount ?? ""}
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
