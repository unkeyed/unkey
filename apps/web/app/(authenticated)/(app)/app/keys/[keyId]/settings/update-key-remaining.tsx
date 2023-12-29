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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { toast } from "@/components/ui/toaster";
import { cn } from "@/lib/utils";
import { updateKeyRemaining } from "./actions";
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    remaining: number | null;
    refillInterval: string | null;
    refillAmount: number | null;
  };
};

export const UpdateKeyRemaining: React.FC<Props> = ({ apiKey }) => {
  const [enabled, setEnabled] = useState(apiKey.remaining !== null);
  const [refillEbabled, setRefillEnabled] = useState(apiKey.remaining !== null);

  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateKeyRemaining(formData);

        if (res.error) {
          toast.error(res.error.message);
          return;
        }

        toast.success("Remaining uses updated");
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
