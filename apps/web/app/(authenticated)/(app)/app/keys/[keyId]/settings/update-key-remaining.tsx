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
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  keyId: z.string(),
  limitEnabled: z.boolean(),
  remaining: z.coerce.number().positive({ message: "Please enter a positive number" }).optional(),
  refill: z
    .object({
      interval: z.enum(["none", "daily", "monthly"]),
      amount: z.coerce
        .number()
        .int()
        .min(1, {
          message: "Please enter the number of uses per interval",
        })
        .positive()
        .optional(),
    })
    .optional(),
});
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    remaining: number | null;
    refillInterval: "daily" | "monthly" | "none";
    refillAmount: number | null;
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
      limitEnabled: apiKey.remaining ? true : false,
      remaining: apiKey.remaining ? apiKey.remaining : undefined,
      refill: {
        interval: apiKey.refillInterval,
        amount: apiKey.refillAmount ? apiKey.refillAmount : undefined,
      },
    },
  });
  const resetLimited = () => {
    // set them to undefined so the form resets properly.
    form.resetField("remaining", undefined);
    form.resetField("refill.amount", undefined);
    form.resetField("refill.interval", { defaultValue: "none" });
    form.resetField("refill", undefined);
  };
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
    if (values.refill?.interval !== "none" && !values.refill?.amount) {
      console.log(values.refill?.amount);

      form.setError("refill.amount", { message: "Please enter the number of uses per interval" });

      return;
    }
    if (values.refill.interval !== "none" && values.remaining === undefined) {
      form.setError("remaining", { message: "Please enter a value" });
      return;
    }
    if (values.limitEnabled === false) {
      delete values.refill;
      delete values.remaining;
    }
    if (values.refill?.interval === "none") {
      delete values.refill;
    }
    updateRemaining.mutateAsync(values);
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
                render={({ field }) => (
                  <FormItem className="">
                    <FormLabel>Refill Rate</FormLabel>
                    <Select
                      onValueChange={field.onChange}
                      defaultValue="none"
                      value={field.value}
                      disabled={!form.watch("limitEnabled") === true}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        <SelectItem value="none">None</SelectItem>
                        <SelectItem value="daily">Daily</SelectItem>
                        <SelectItem value="monthly">Monthly</SelectItem>
                      </SelectContent>
                    </Select>
                    <FormMessage />
                  </FormItem>
                )}
              />
              <FormField
                control={form.control}
                disabled={
                  form.watch("refill.interval") === "none" ||
                  form.watch("refill.interval") === undefined ||
                  !form.watch("limitEnabled") === true
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
                        value={form.watch("refill.interval") === "none" ? undefined : field.value}
                      />
                    </FormControl>
                    <FormDescription>
                      Enter the number of uses to refill per interval.
                    </FormDescription>
                    <FormMessage defaultValue="Please enter a value if interval is selected" />
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
            <SubmitButton label="Save" />
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
