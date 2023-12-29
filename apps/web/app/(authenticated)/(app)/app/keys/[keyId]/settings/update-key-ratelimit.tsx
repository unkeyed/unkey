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
import { Switch } from "@/components/ui/switch";
import { toast } from "@/components/ui/toaster";
import { cn } from "@/lib/utils";
import { Key } from "@unkey/db";
import React, { useState } from "react";
import { updateKeyRatelimit } from "./actions";
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    ratelimitType: Key["ratelimitType"];
    ratelimitLimit: number | null;
    ratelimitRefillRate: number | null;
    ratelimitRefillInterval: number | null;
  };
};

export const UpdateKeyRatelimit: React.FC<Props> = ({ apiKey }) => {
  const [enabled, setEnabled] = useState(apiKey.ratelimitType !== null);
  return (
    <form
      action={async (formData: FormData) => {
        const res = await updateKeyRatelimit(formData);
        if (res.error) {
          toast.error(res.error.message);
          return;
        }

        toast.success("Ratelimit updated");
      }}
    >
      <Card>
        <CardHeader>
          <CardTitle>Ratelimit</CardTitle>
          <CardDescription>How frequently this key can be used.</CardDescription>
        </CardHeader>
        <CardContent className="relative flex justify-between item-center">
          <div className={cn("flex flex-col", { "opacity-50": !enabled })}>
            <input type="hidden" name="keyId" value={apiKey.id} />
            <input type="hidden" name="enabled" value={enabled ? "true" : "false"} />
            <input type="hidden" name="ratelimitType" value={enabled ? "fast" : undefined} />

            <div className="flex flex-col gap-1">
              <Label htmlFor="ratelimitLimit">Limit</Label>
              <Input
                disabled={!enabled}
                type="number"
                min={0}
                name="ratelimitLimit"
                className="max-w-sm"
                defaultValue={apiKey.ratelimitLimit ?? undefined}
                autoComplete="off"
              />
              <p className="mt-1 text-xs text-content-subtle">
                The maximum number of requests possible during a burst.
              </p>
            </div>
            <div className="flex items-center justify-between w-full gap-4 mt-8">
              <div className="flex flex-col gap-1">
                <Label htmlFor="ratelimitRefillRate">Refill Rate</Label>

                <Input
                  disabled={!enabled}
                  type="number"
                  min={0}
                  name="ratelimitRefillRate"
                  className="max-w-sm"
                  defaultValue={apiKey.ratelimitRefillRate ?? undefined}
                  autoComplete="off"
                />
              </div>
              <div className="flex flex-col gap-1">
                <Label htmlFor="ratelimitRefillInterval">
                  Refill Interval{" "}
                  <span className="text-xs text-content-subtle">(milliseconds)</span>
                </Label>

                <Input
                  disabled={!enabled}
                  type="number"
                  min={0}
                  name="ratelimitRefillInterval"
                  className="max-w-sm"
                  defaultValue={apiKey.ratelimitRefillInterval ?? undefined}
                  autoComplete="off"
                />
              </div>
            </div>
            <p className="mt-1 text-xs text-content-subtle">
              How many requests may be performed in a given interval
            </p>
          </div>
        </CardContent>
        <CardFooter className="justify-between">
          <div className="flex items-center gap-4">
            <Switch id="enabled" checked={enabled} onCheckedChange={setEnabled} />
            <Label htmlFor="enabled">{enabled ? "Enabled" : "Disabled"}</Label>
          </div>
          <SubmitButton label="Save" />
        </CardFooter>
      </Card>
    </form>
  );
};
