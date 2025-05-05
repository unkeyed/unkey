"use client";

import { Loading } from "@/components/dashboard/loading";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  keyId: z.string(),
  limitEnabled: z.boolean(),
  remaining: z.coerce.number().positive({ message: "Please enter a positive number" }).optional(),
  refill: z
    .object({
      interval: z.enum(["none", "daily", "monthly"]).optional(),
      amount: z
        .literal("")
        .transform(() => undefined)
        .or(z.coerce.number().int().positive())
        .optional(),
      refillDay: z
        .literal("")
        .transform(() => undefined)
        .or(z.coerce.number().int().max(31).positive())
        .optional(),
    })
    .optional(),
});
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    remaining: number | null;
    refillAmount: number | null;
    refillDay: number | null | undefined;
  };
};

export const UpdateKeyRemaining: React.FC<Props> = ({ apiKey }) => {
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      keyId: apiKey.id,
      limitEnabled: Boolean(apiKey.remaining) || Boolean(apiKey.refillAmount),
      remaining: apiKey.remaining ? apiKey.remaining : undefined,
      refill: {
        interval: apiKey.refillDay ? "monthly" : apiKey.refillAmount ? "daily" : "none",
        amount: apiKey.refillAmount ? apiKey.refillAmount : undefined,
        refillDay: apiKey.refillDay ?? undefined,
      },
    },
  });
  const resetLimited = () => {
    // set them to undefined so the form resets properly.
    form.resetField("remaining", undefined);
    form.resetField("refill.amount", undefined);
    form.resetField("refill.refillDay", undefined);
    form.resetField("refill", undefined);
  };
  const updateRemaining = trpc.key.update.remaining.useMutation({
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
    if (values.refill?.interval === "none") {
      delete values.refill;
    }
    // make sure they aren't sent to the server if they are disabled.
    if (
      values.refill?.interval !== undefined &&
      values.refill?.interval !== "none" &&
      !values.refill?.amount
    ) {
      form.setError("refill.amount", {
        type: "manual",
        message: "Please enter a value if interval is selected",
      });
      return;
    }
    if (!values.refill?.amount && values.refill?.refillDay && values.limitEnabled) {
      form.setError("refill.amount", {
        message: "Please enter the number of uses per interval or remove the refill day",
      });
      return;
    }
    if (values.remaining === undefined && values.limitEnabled) {
      form.setError("remaining", { message: "Please enter a value" });
      return;
    }
    if (!values.limitEnabled) {
      delete values.refill;
      delete values.remaining;
    }
    if (values.refill?.interval === "monthly" && !values.refill?.refillDay) {
      values.refill.refillDay = 1;
    }
    await updateRemaining.mutateAsync({
      keyId: values.keyId,
      limitEnabled: values.limitEnabled,
      remaining: values.remaining,
      refill: values.refill,
    });
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
                "opacity-50": !form.getValues("limitEnabled"),
              })}
            >
              <Label htmlFor="remaining">Remaining</Label>
              <FormField
                control={form.control}
                name="remaining"
                disabled={!form.watch("limitEnabled") === true}
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Number of uses</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="100"
                        className="w-full"
                        type="number"
                        {...field}
                        value={field.value}
                      />
                    </FormControl>
                    <FormDescription>
                      Enter the remaining amount of uses for this key.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                name="refill.interval"
                disabled={
                  !form.watch("limitEnabled") ||
                  form.watch("remaining") === undefined ||
                  form.watch("remaining") === null
                }
                render={({ field }) => (
                  <FormItem className="">
                    <FormLabel>Refill Rate</FormLabel>
                    <Select onValueChange={field.onChange} value={field.value || "none"}>
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">None</SelectItem>
                        <SelectItem value="daily">Daily</SelectItem>
                        <SelectItem value="monthly">Monthly</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormDescription>Interval key will be refilled.</FormDescription>
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                disabled={
                  form.watch("remaining") === undefined ||
                  form.watch("remaining") === null ||
                  !form.watch("limitEnabled") === true ||
                  form.watch("refill.interval") === "none"
                }
                name="refill.amount"
                render={({ field }) => (
                  <FormItem className="mt-4">
                    <FormLabel>Number of uses per interval</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="100"
                        className="w-full"
                        type="number"
                        {...field}
                        value={form.getValues("limitEnabled") ? field.value : undefined}
                      />
                    </FormControl>
                    <FormDescription>
                      Enter the number of uses to refill per interval.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                disabled={
                  form.watch("remaining") === undefined ||
                  form.watch("remaining") === null ||
                  !form.watch("limitEnabled") === true ||
                  form.watch("refill.interval") === "none" ||
                  form.watch("refill.interval") === "daily"
                }
                name="refill.refillDay"
                render={({ field }) => (
                  <FormItem className="mt-4">
                    <FormLabel>Day of the month to refill uses.</FormLabel>
                    <FormControl>
                      <Input
                        placeholder="1"
                        className="w-full"
                        type="number"
                        {...field}
                        value={
                          form.getValues("limitEnabled") &&
                          form.getValues("refill.interval") === "monthly"
                            ? field.value
                            : undefined
                        }
                      />
                    </FormControl>
                    <FormDescription>
                      Enter the day to refill monthly or leave blank for daily refill
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </CardContent>
          <CardFooter className="justify-between">
            <FormField
              control={form.control}
              name="limitEnabled"
              render={({ field }) => (
                <FormItem className="w-full">
                  <div className="flex items-center gap-4">
                    <FormControl>
                      <Switch
                        checked={form.getValues("limitEnabled")}
                        onCheckedChange={(e) => {
                          field.onChange(e);
                          resetLimited();
                        }}
                      />
                    </FormControl>{" "}
                    <FormLabel htmlFor="limitEnabled">
                      {form.getValues("limitEnabled") ? "Enabled" : "Disabled"}
                    </FormLabel>
                  </div>
                </FormItem>
              )}
            />
            <Button disabled={updateRemaining.isLoading || !form.formState.isValid} type="submit">
              {updateRemaining.isLoading ? <Loading /> : "Save"}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
