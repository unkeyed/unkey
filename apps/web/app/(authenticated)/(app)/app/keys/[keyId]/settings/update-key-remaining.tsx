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
import { FormField } from "@/components/ui/form";
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
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import React, { useState } from "react";
import { Form, useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  keyId: z.string(),
  enableRemaining: z.boolean(),
  remaining: z.number().int().optional(),
  refillInterval: z.enum(["daily", "monthly", "null"]).optional(),
  refillAmount: z.number().int().optional(),
});

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
  const router = useRouter();
  const [enabled, setEnabled] = useState(apiKey.remaining !== null);
  const [refillEbabled, setRefillEnabled] = useState(apiKey.remaining !== null);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
  });

  const updateRemaining = trpc.keySettings.updateRemaining.useMutation({
    onSuccess() {
      toast.success("Remaining uses has updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    updateRemaining.mutate(values);
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
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
              <FormField
                control={form.control}
                name="remaining"
                render={({ field }) => (
                  <Input
                    {...field}
                    disabled={!enabled}
                    type="number"
                    min={0}
                    className="max-w-sm"
                    defaultValue={apiKey?.remaining ?? ""}
                    autoComplete="off"
                  />
                )}
              />

              <Label htmlFor="refillInterval" className="pt-4">
                Refill Rate
              </Label>
              <FormField
                control={form.control}
                name="refillInterval"
                render={({ field }) => (
                  <Select
                    {...field}
                    disabled={!enabled}
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
                )}
              />
              <FormField
                control={form.control}
                name="refillAmount"
                render={({ field }) => (
                  <Input
                    {...field}
                    disabled={!refillEbabled}
                    placeholder="100"
                    className="w-full"
                    min={1}
                    type="number"
                    defaultValue={apiKey?.refillAmount ?? ""}
                  />
                )}
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
    </Form>
  );
};
