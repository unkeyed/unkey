"use client";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Loading } from "@/components/dashboard/loading";
import { VisibleButton } from "@/components/dashboard/visible-button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
import { Code } from "@/components/ui/code";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { AlertCircle } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const getDatePlusTwoMinutes = () => {
  const now = new Date();
  const futureDate = new Date(now.getTime() + 2 * 60000);
  return futureDate.toISOString().slice(0, -8);
};
const formSchema = z.object({
  bytes: z.coerce
    .number({
      errorMap: (issue, { defaultError }) => ({
        message:
          issue.code === "invalid_type"
            ? "Amount must be a number and greater than 0"
            : defaultError,
      }),
    })
    .default(16),
  prefix: z
    .string()
    .max(8, { message: "Please limit the prefix to under 8 characters." })
    .optional(),
  ownerId: z.string().optional(),
  name: z.string().optional(),
  metaEnabled: z.boolean().default(false),
  meta: z
    .string()
    .refine(
      (s) => {
        try {
          JSON.parse(s);
          return true;
        } catch {
          return false;
        }
      },
      {
        message: "Must be valid json",
      },
    )
    .optional(),
  limitEnabled: z.boolean().default(false),
  limit: z
    .object({
      remaining: z.coerce
        .number({
          errorMap: (issue, { defaultError }) => ({
            message:
              issue.code === "invalid_type"
                ? "Remaining amount must be greater than 0"
                : defaultError,
          }),
        })
        .int()
        .positive({ message: "Please enter a positive number" })
        .optional(),
      refill: z
        .object({
          interval: z.enum(["none", "daily", "monthly"]).default("none"),
          amount: z.coerce
            .number({
              errorMap: (issue, { defaultError }) => ({
                message:
                  issue.code === "invalid_type"
                    ? "Refill amount must be greater than 0 and a integer"
                    : defaultError,
              }),
            })
            .int()
            .min(1)
            .positive()
            .optional(),
        })
        .optional(),
    })
    .optional(),
  expireEnabled: z.boolean().default(false),
  expires: z.coerce
    .date()
    .min(new Date(new Date().getTime() + 2 * 60000))
    .optional(),
  ratelimitEnabled: z.boolean().default(false),
  ratelimit: z
    .object({
      type: z.enum(["consistent", "fast"]).default("fast"),
      refillInterval: z.coerce
        .number({
          errorMap: (issue, { defaultError }) => ({
            message:
              issue.code === "invalid_type"
                ? "Refill interval must be greater than 0"
                : defaultError,
          }),
        })
        .positive({ message: "Refill interval must be greater than 0" }),
      refillRate: z.coerce
        .number({
          errorMap: (issue, { defaultError }) => ({
            message:
              issue.code === "invalid_type" ? "Refill rate must be greater than 0" : defaultError,
          }),
        })
        .positive({ message: "Refill rate must be greater than 0" }),
      limit: z.coerce
        .number({
          errorMap: (issue, { defaultError }) => ({
            message:
              issue.code === "invalid_type" ? "Refill limit must be greater than 0" : defaultError,
          }),
        })
        .positive({ message: "Limit must be greater than 0" }),
    })
    .optional(),
});

type Props = {
  apiId: string;
};

export const CreateKey: React.FC<Props> = ({ apiId }) => {
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: async (data, context, options) => {
      return zodResolver(formSchema)(data, context, options);
    },
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      bytes: 16,
      expireEnabled: false,
      limitEnabled: false,
      metaEnabled: false,
      ratelimitEnabled: false,
    },
  });

  const key = trpc.key.create.useMutation({
    onSuccess() {
      toast("Key Created", {
        description: "Your Key has been created",
      });
    },
    onError(_err) {
      toast.error("An error occured, please try again");
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    // make sure they aren't sent to the server if they are disabled.
    if (!values.expireEnabled) {
      delete values.expires;
    }
    if (!values.metaEnabled) {
      delete values.meta;
    }
    if (!values.limitEnabled) {
      delete values.limit;
    }
    if (!values.ratelimitEnabled) {
      delete values.ratelimit;
    }

    await key.mutateAsync({
      apiId,
      ...values,
      meta: values.meta ? JSON.parse(values.meta) : undefined,
      expires: values.expires?.getTime() ?? undefined,
      ownerId: values.ownerId ?? undefined,
      remaining: values.limit?.remaining ?? undefined,
      enabled: true,
    });
  }

  const snippet = `curl -XPOST '${process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "https://api.unkey.dev"}/v1/keys/verify' \\
  -H 'Content-Type: application/json' \\
  -d '{
    "key": "${key.data?.key}"
  }'`;

  const split = key.data?.key.split("_") ?? [];
  const maskedKey =
    split.length >= 2
      ? `${split.at(0)}_${"*".repeat(split.at(1)?.length ?? 0)}`
      : "*".repeat(split.at(0)?.length ?? 0);
  const [showKey, setShowKey] = useState(false);
  const [showKeyInSnippet, setShowKeyInSnippet] = useState(false);

  const resetRateLimit = () => {
    // set them to undefined so the form resets properly.
    form.resetField("ratelimit.refillRate", undefined);
    form.resetField("ratelimit.refillInterval", undefined);
    form.resetField("ratelimit.limit", undefined);
    form.resetField("ratelimit", undefined);
  };

  const resetLimited = () => {
    form.resetField("limit.refill.amount", undefined);
    form.resetField("limit.refill.interval", undefined);
    form.resetField("limit.refill", undefined);
    form.resetField("limit.remaining", undefined);
    form.resetField("limit", undefined);
  };

  useEffect(() => {
    // React hook form + zod doesn't play nice with nested objects, so we need to reset them on load.
    resetRateLimit();
    resetLimited();
  }, []);

  return (
    <>
      {key.data ? (
        <div className="w-full max-sm:p-4">
          <div>
            <p className="mb-4 text-xl font-bold">Your API Key</p>
            <Alert>
              <AlertCircle className="w-4 h-4" />
              <AlertTitle>This key is only shown once and can not be recovered </AlertTitle>
              <AlertDescription>
                Please pass it on to your user or store it somewhere safe.
              </AlertDescription>
            </Alert>

            <Code className="flex items-center justify-between w-full gap-4 my-8 ph-no-capture max-sm:text-xs sm:overflow-hidden">
              <pre>{showKey ? key.data.key : maskedKey}</pre>
              <div className="flex items-start justify-between gap-4 max-sm:absolute max-sm:right-11">
                <VisibleButton isVisible={showKey} setIsVisible={setShowKey} />
                <CopyButton value={key.data.key} />
              </div>
            </Code>
          </div>

          <p className="my-2 font-medium text-center text-gray-700 ">Try verifying it:</p>
          <Code className="flex items-start justify-between w-full gap-4 my-8 overflow-hidden max-sm:text-xs ">
            <div className="max-sm:mt-10">
              <pre className="ph-no-capture">
                {showKeyInSnippet ? snippet : snippet.replace(key.data.key, maskedKey)}
              </pre>
            </div>
            <div className="flex items-start justify-between gap-4 max-ms:top-2 max-sm:absolute max-sm:right-11 ">
              <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
              <CopyButton value={snippet} />
            </div>
          </Code>
          <div className="flex justify-end my-4 space-x-4">
            <Link href={`/app/apis/${apiId}`}>
              <Button variant="secondary">Back</Button>
            </Link>
            <Button
              onClick={() => {
                key.reset();
                form.setValue("expireEnabled", false);
                form.setValue("ratelimitEnabled", false);
                form.setValue("metaEnabled", false);
                form.setValue("limitEnabled", false);
                router.refresh();
              }}
            >
              Create another key
            </Button>
          </div>
        </div>
      ) : (
        <>
          <div>
            <div className="w-full">
              <Form {...form}>
                <form
                  className="flex flex-col h-full gap-8 md:flex-row"
                  onSubmit={form.handleSubmit(onSubmit)}
                >
                  <div className="z-0 flex flex-col w-full h-full gap-4 md:sticky top-24 md:w-1/2">
                    <h2 className="text-2xl font-semibold tracking-tight">Create a new key</h2>
                    <FormField
                      control={form.control}
                      name="prefix"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Prefix</FormLabel>
                          <FormControl>
                            <Input
                              {...field}
                              onBlur={(e) => {
                                if (e.target.value === "") {
                                  return;
                                }
                              }}
                            />
                          </FormControl>
                          <FormDescription>
                            Using a prefix can make it easier for your users to distinguish between
                            apis. Don't add a trailing underscore, we'll do that automatically.
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="bytes"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Bytes</FormLabel>
                          <FormControl>
                            <Input type="number" {...field} />
                          </FormControl>
                          <FormDescription>How many bytes to use.</FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="ownerId"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Owner</FormLabel>
                          <FormControl>
                            <Input {...field} />
                          </FormControl>
                          <FormDescription>
                            This is the id of the user or workspace in your system, so you can
                            identify users from an API key.
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="name"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>Name</FormLabel>
                          <FormControl>
                            <Input {...field} />
                          </FormControl>
                          <FormDescription>
                            To make it easier to identify a particular key, you can provide a name.
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                  </div>

                  <Separator orientation="vertical" className="" />

                  <div className="flex flex-col w-full gap-4 md:w-1/2">
                    <Card>
                      <CardContent className="justify-between w-full p-4 item-center">
                        <div className="flex items-center justify-between w-full">
                          <span>Ratelimit</span>

                          <FormField
                            control={form.control}
                            name="ratelimitEnabled"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel className="sr-only">Ratelimit</FormLabel>
                                <FormControl>
                                  <Switch
                                    onCheckedChange={(e) => {
                                      field.onChange(e);
                                      if (field.value === false) {
                                        resetRateLimit();
                                      }
                                    }}
                                  />
                                </FormControl>
                              </FormItem>
                            )}
                          />
                        </div>

                        {form.watch("ratelimitEnabled") ? (
                          <>
                            <div className="flex flex-col gap-4 mt-4">
                              <FormField
                                control={form.control}
                                name="ratelimit.limit"
                                render={({ field }) => (
                                  <FormItem>
                                    <FormLabel>Limit</FormLabel>
                                    <FormControl>
                                      <Input
                                        placeholder="10"
                                        {...field}
                                        value={
                                          form.getValues("ratelimitEnabled")
                                            ? field.value
                                            : undefined
                                        }
                                      />
                                    </FormControl>
                                    <FormDescription>
                                      The maximum number of requests possible during a burst.
                                    </FormDescription>
                                    <FormMessage />
                                  </FormItem>
                                )}
                              />
                              <FormField
                                control={form.control}
                                name="ratelimit.refillRate"
                                render={({ field }) => (
                                  <FormItem>
                                    <FormLabel>Refill Rate</FormLabel>
                                    <FormControl>
                                      <Input placeholder="5" type="number" {...field} />
                                    </FormControl>

                                    <FormMessage />
                                  </FormItem>
                                )}
                              />
                              <FormField
                                control={form.control}
                                name="ratelimit.refillInterval"
                                render={({ field }) => (
                                  <FormItem>
                                    <FormLabel>Refill Interval (milliseconds)</FormLabel>
                                    <FormControl>
                                      <Input
                                        placeholder="1000"
                                        type="number"
                                        {...field}
                                        value={
                                          form.getValues("ratelimitEnabled")
                                            ? field.value
                                            : undefined
                                        }
                                      />
                                    </FormControl>

                                    <FormMessage />
                                  </FormItem>
                                )}
                              />
                              <FormDescription>
                                How many requests may be performed in a given interval
                              </FormDescription>
                            </div>
                            {form.formState.errors.ratelimit && (
                              <p className="text-xs text-center text-content-alert">
                                {form.formState.errors.ratelimit.message}
                              </p>
                            )}
                          </>
                        ) : null}
                      </CardContent>
                    </Card>

                    <Card>
                      <CardContent className="justify-between w-full p-4 item-center">
                        <div className="flex items-center justify-between w-full">
                          <span>Limited Use</span>

                          <FormField
                            control={form.control}
                            name="limitEnabled"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel className="sr-only">Limited Use</FormLabel>
                                <FormControl>
                                  <Switch
                                    onCheckedChange={(e) => {
                                      field.onChange(e);
                                      if (field.value === false) {
                                        resetLimited();
                                      }
                                    }}
                                  />
                                </FormControl>
                              </FormItem>
                            )}
                          />
                        </div>

                        {form.watch("limitEnabled") ? (
                          <>
                            <p className="text-xs text-content-subtle">
                              How many times this key can be used before it gets disabled
                              automatically.
                            </p>
                            <div className="flex flex-col gap-4 mt-4">
                              <FormField
                                control={form.control}
                                name="limit.remaining"
                                render={({ field }) => (
                                  <FormItem>
                                    <FormLabel>Number of uses</FormLabel>
                                    <FormControl>
                                      <Input
                                        placeholder="100"
                                        className="w-full"
                                        type="number"
                                        {...field}
                                        value={
                                          form.getValues("limitEnabled") ? field.value : undefined
                                        }
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
                                name="limit.refill.interval"
                                render={({ field }) => (
                                  <FormItem className="">
                                    <FormLabel>Refill Rate</FormLabel>
                                    <Select
                                      onValueChange={field.onChange}
                                      defaultValue="none"
                                      value={field.value}
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
                                  form.watch("limit.refill.interval") === "none" ||
                                  form.watch("limit.refill.interval") === undefined
                                }
                                name="limit.refill.amount"
                                render={({ field }) => (
                                  <FormItem className="mt-4">
                                    <FormLabel>Number of uses per interval</FormLabel>
                                    <FormControl>
                                      <Input
                                        placeholder="100"
                                        className="w-full"
                                        type="number"
                                        {...field}
                                        value={
                                          form.getValues("limitEnabled") ? field.value : undefined
                                        }
                                      />
                                    </FormControl>
                                    <FormDescription>
                                      Enter the number of uses to refill per interval.
                                    </FormDescription>
                                    <FormMessage defaultValue="Please enter a value if interval is selected" />
                                  </FormItem>
                                )}
                              />
                              <FormDescription>
                                How many requests may be performed in a given interval
                              </FormDescription>
                            </div>
                            {form.formState.errors.ratelimit && (
                              <p className="text-xs text-center text-content-alert">
                                {form.formState.errors.ratelimit.message}
                              </p>
                            )}
                          </>
                        ) : null}
                      </CardContent>
                    </Card>
                    <Card>
                      <CardContent className="justify-between w-full p-4 item-center">
                        <div className="flex items-center justify-between w-full">
                          <span>Expiration</span>

                          <FormField
                            control={form.control}
                            name="expireEnabled"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel className="sr-only">Expiration</FormLabel>
                                <FormControl>
                                  <Switch
                                    onCheckedChange={(e) => {
                                      field.onChange(e);
                                      if (field.value === false) {
                                        resetLimited();
                                      }
                                    }}
                                  />
                                </FormControl>
                              </FormItem>
                            )}
                          />
                        </div>

                        {form.watch("expireEnabled") ? (
                          <>
                            <p className="text-xs text-content-subtle">
                              {" "}
                              Automatically revoke this key after a certain date.
                            </p>
                            <div className="flex flex-col gap-4 mt-4">
                              <FormField
                                control={form.control}
                                name="expires"
                                render={({ field }) => (
                                  <FormItem>
                                    <FormLabel>Expiry Date</FormLabel>
                                    <FormControl>
                                      <Input
                                        type="datetime-local"
                                        {...field}
                                        defaultValue={getDatePlusTwoMinutes()}
                                        value={
                                          form.getValues("expireEnabled")
                                            ? field.value?.toLocaleString()
                                            : undefined
                                        }
                                      />
                                    </FormControl>
                                    <FormDescription>
                                      This api key will automatically be revoked after the given
                                      date.
                                    </FormDescription>
                                    <FormMessage />
                                  </FormItem>
                                )}
                              />
                              <FormDescription>
                                How many requests may be performed in a given interval
                              </FormDescription>
                            </div>
                            {form.formState.errors.ratelimit && (
                              <p className="text-xs text-center text-content-alert">
                                {form.formState.errors.ratelimit.message}
                              </p>
                            )}
                          </>
                        ) : null}
                      </CardContent>
                    </Card>
                    <Card>
                      <CardContent className="justify-between w-full p-4 item-center">
                        <div className="flex items-center justify-between w-full">
                          <span>Metadata</span>

                          <FormField
                            control={form.control}
                            name="metaEnabled"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel className="sr-only">Metadata</FormLabel>
                                <FormControl>
                                  <Switch
                                    onCheckedChange={(e) => {
                                      field.onChange(e);
                                      if (field.value === false) {
                                        resetLimited();
                                      }
                                    }}
                                  />
                                </FormControl>
                              </FormItem>
                            )}
                          />
                        </div>

                        {form.watch("metaEnabled") ? (
                          <>
                            <p className="text-xs text-content-subtle">
                              Store json, or any other data you want to associate with this key.
                              Whenever you verify this key, we'll return the metadata to you. Enter
                              custom metadata as a JSON object.Format Json
                            </p>

                            <div className="flex flex-col gap-4 mt-4">
                              <FormField
                                control={form.control}
                                name="meta"
                                render={({ field }) => (
                                  <FormItem>
                                    <FormControl>
                                      <Textarea
                                        disabled={!form.watch("metaEnabled")}
                                        className="m-4 mx-auto border rounded-md shadow-sm"
                                        rows={7}
                                        placeholder={`{"stripeCustomerId" : "cus_9s6XKzkNRiz8i3"}`}
                                        {...field}
                                        value={
                                          form.getValues("metaEnabled") ? field.value : undefined
                                        }
                                      />
                                    </FormControl>
                                    <FormDescription>
                                      Enter custom metadata as a JSON object.
                                    </FormDescription>
                                    <FormMessage />
                                    <Button
                                      variant="secondary"
                                      type="button"
                                      onClick={(_e) => {
                                        try {
                                          if (field.value) {
                                            const parsed = JSON.parse(field.value);
                                            field.onChange(JSON.stringify(parsed, null, 2));
                                            form.clearErrors("meta");
                                          }
                                        } catch (_e) {
                                          form.setError("meta", {
                                            type: "manual",
                                            message: "Invalid JSON",
                                          });
                                        }
                                      }}
                                      value={field.value}
                                    >
                                      Format Json
                                    </Button>
                                  </FormItem>
                                )}
                              />
                            </div>
                            {form.formState.errors.ratelimit && (
                              <p className="text-xs text-center text-content-alert">
                                {form.formState.errors.ratelimit.message}
                              </p>
                            )}
                          </>
                        ) : null}
                      </CardContent>
                    </Card>
                    <div className="w-full">
                      <Button
                        className="w-full"
                        disabled={key.isLoading}
                        type="submit"
                        variant={key.isLoading || !form.formState.isValid ? "disabled" : "primary"}
                      >
                        {key.isLoading ? <Loading /> : "Create"}
                      </Button>
                    </div>
                  </div>
                </form>
              </Form>
            </div>
          </div>
        </>
      )}
    </>
  );
};
