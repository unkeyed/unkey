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
import { useToast } from "@/components/ui/use-toast";
import { cn } from "@/lib/utils";
import { updateKeyRemaining } from "./actions";
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    remaining: number | null;
    refillInterval: number | null;
    refillIncrement: number | null;
  };
};

export const UpdateKeyRemaining: React.FC<Props> = ({ apiKey }) => {
  const { toast } = useToast();

  const [enabled, setEnabled] = useState(apiKey.remaining !== null);
  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateKeyRemaining(formData);
        console.log(formData);
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
              defaultValue={apiKey.remaining ?? ""}
              autoComplete="off"
            />
            <Label htmlFor="replenish" className="pt-4">
              Replenishment Rate
            </Label>
            <Select
              disabled={!enabled}
              name="refillInterval"
              defaultValue={apiKey?.refillInterval ? apiKey?.refillInterval.toString() : "None"}
            >
              <SelectTrigger>
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                <SelectItem value="0">None</SelectItem>
                <SelectItem value="86400000">Daily</SelectItem>
                <SelectItem value="604800000">Weekly</SelectItem>
                <SelectItem value="1209600000">Bi-Weekly</SelectItem>
                <SelectItem value="2629800000">Monthly</SelectItem>
              </SelectContent>
            </Select>
            <Input
              name="refillIncrement"
              placeholder="100"
              className="w-full"
              type="number"
              defaultValue={apiKey.refillIncrement ? apiKey.refillIncrement : "100"}
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
