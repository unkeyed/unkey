"use client";
import { revalidate } from "@/app/actions";
import { CopyButton } from "@/components/dashboard/copy-button";
import { Loading } from "@/components/dashboard/loading";
import { VisibleButton } from "@/components/dashboard/visible-button";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Badge } from "@/components/ui/badge";
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
import { Button } from "@unkey/ui";

import { Separator } from "@/components/ui/separator";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { addMinutes, format } from "date-fns";
import { AlertCircle } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import type { z } from "zod";
import { formSchema } from "./validation";

export const dynamic = "force-dynamic";

type Props = {
  apiId: string;
  keyAuthId: string;
  defaultBytes: number | null;
  defaultPrefix: string | null;
};

export const CreateKey = ({ apiId, keyAuthId, defaultBytes, defaultPrefix }: Props) => {
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: async (data, context, options) => {
      return zodResolver(formSchema)(data, context, options);
    },
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    // Should required to unregister form elements when they are not rendered.
    shouldUnregister: true,
    defaultValues: {
      prefix: defaultPrefix || undefined,
      bytes: defaultBytes || 16,
      expireEnabled: false,
      limitEnabled: false,
      metaEnabled: false,
      ratelimitEnabled: false,
      limit: {
        remaining: undefined,
        refill: {
          interval: "none",
          amount: undefined,
          refillDay: undefined,
        },
      },
    },
  });

  const key = trpc.key.create.useMutation({
    onSuccess() {
      toast.success("Key Created", {
        description: "Your Key has been created",
      });
      revalidate(`/keys/${keyAuthId}`);
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (
      values.limitEnabled &&
      values.limit?.refill?.interval !== "none" &&
      !values.limit?.refill?.amount
    ) {
      form.setError("limit.refill.amount", {
        type: "manual",
        message: "Please enter a value if interval is selected",
      });
      return;
    }

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
    const refill = values.limit?.refill;
    if (refill?.interval === "daily") {
      refill.refillDay = undefined;
    }
    if (refill?.interval === "monthly" && !refill.refillDay) {
      refill.refillDay = 1;
    }
    await key.mutateAsync({
      keyAuthId,
      ...values,
      meta: values.meta ? JSON.parse(values.meta) : undefined,
      expires: values.expires?.getTime() ?? undefined,
      ownerId: values.ownerId ?? undefined,
      remaining: values.limit?.remaining ?? undefined,
      refill:
        refill?.amount && refill.interval !== "none"
          ? {
              amount: refill.amount,
              refillDay: refill.interval === "daily" ? null : refill.refillDay ?? 1,
            }
          : undefined,
      enabled: true,
    });

    router.refresh();
  }

  const snippet = `curl -XPOST '${process.env.NEXT_PUBLIC_UNKEY_API_URL ?? "https://api.unkey.dev"}/v1/keys.verifyKey' \\
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
    form.resetField("ratelimit.duration", undefined);
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

  // biome-ignore lint/correctness/useExhaustiveDependencies: reset is only required on mount
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
            <div className="flex flex-col sm:flex-row justify-between sm:items-center mb-4">
              <p className="mb-4 sm:mb-0 text-xl font-bold">Your API Key</p>
              <Code className="h-8 w-full sm:w-auto flex gap-1.5 justify-between">
                <pre className="truncate">{key.data.keyId}</pre>
                <CopyButton value={key.data.keyId} />
              </Code>
            </div>
            <Alert>
              <AlertCircle className="w-4 h-4" />
              <AlertTitle>This key is only shown once and can not be recovered </AlertTitle>
              <AlertDescription>
                Please pass it on to your user or store it somewhere safe.
              </AlertDescription>
            </Alert>
            <Code className="flex items-center justify-between w-full gap-4 mt-2 my-8 ph-no-capture max-sm:text-xs sm:overflow-hidden">
              <pre className="overflow-x-auto">{showKey ? key.data.key : maskedKey}</pre>
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
            <div className="flex items-center justify-between gap-4 max-ms:top-2 max-sm:absolute max-sm:right-11 ">
              <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
              <CopyButton value={snippet} />
            </div>
          </Code>
          <div className="flex justify-end my-4 space-x-4">
            <Link href={`/keys/${keyAuthId}`}>
              <Button>Back</Button>
            </Link>
            <Link href={`/apis/${apiId}/keys/${keyAuthId}/${key.data.keyId}`}>
              <Button>View key details</Button>
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
              <h2 className="text-2xl font-semibold tracking-tight">Create a new key</h2>
              <Form {...form}>
                <form
                  className="flex flex-col h-full gap-8 mt-4 md:flex-row"
                  onSubmit={form.handleSubmit(onSubmit)}
                >
                  <div className="z-0 flex flex-col w-full h-full gap-4 md:sticky top-24 md:w-1/2">
                    <FormField
                      control={form.control}
                      name="prefix"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            Prefix{" "}
                            <Badge variant="secondary" size="sm">
                              Optional
                            </Badge>
                          </FormLabel>
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
                            apis. Don't add a trailing underscore, we'll do that automatically:{" "}
                            <span className="font-mono font-light">{"<prefix>_randombytes"}</span>
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
                          <FormLabel>
                            Bytes{" "}
                            <Badge variant="secondary" size="sm">
                              Optional
                            </Badge>
                          </FormLabel>
                          <FormControl>
                            <Input type="number" {...field} />
                          </FormControl>
                          <FormDescription>
                            How long the key will be. Longer keys are harder to guess and more
                            secure.
                          </FormDescription>
                          <FormMessage />
                        </FormItem>
                      )}
                    />
                    <FormField
                      control={form.control}
                      name="ownerId"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            Owner{" "}
                            <Badge variant="secondary" size="sm">
                              Optional
                            </Badge>
                          </FormLabel>
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
                          <FormLabel>
                            Name{" "}
                            <Badge variant="secondary" size="sm">
                              Optional
                            </Badge>
                          </FormLabel>
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
                    <FormField
                      control={form.control}
                      name="environment"
                      render={({ field }) => (
                        <FormItem>
                          <FormLabel>
                            Environment{" "}
                            <Badge variant="secondary" size="sm">
                              Optional
                            </Badge>
                          </FormLabel>
                          <FormControl>
                            <Input {...field} />
                          </FormControl>
                          <FormDescription>
                            Separate keys into different environments, for example{" "}
                            <strong>test</strong> and <strong>live</strong>.
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
                                      The maximum number of requests in the given fixed window.
                                    </FormDescription>
                                    <FormMessage />
                                  </FormItem>
                                )}
                              />

                              <FormField
                                control={form.control}
                                name="ratelimit.duration"
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
                                    <FormDescription>
                                      The time window in milliseconds for the rate limit to reset.
                                    </FormDescription>
                                    <FormMessage />
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
                                    <Select onValueChange={field.onChange} value={field.value}>
                                      <SelectTrigger>
                                        <SelectValue />
                                      </SelectTrigger>
                                      <SelectContent>
                                        <SelectItem value="none">None</SelectItem>
                                        <SelectItem value="daily">Daily</SelectItem>
                                        <SelectItem value="monthly">Monthly</SelectItem>
                                      </SelectContent>
                                    </Select>
                                    <FormDescription>
                                      Interval key will be refilled.
                                    </FormDescription>
                                  </FormItem>
                                )}
                              />
                              <FormField
                                control={form.control}
                                disabled={
                                  !form.watch("limitEnabled") ||
                                  form.watch("limit.refill.interval") === "none"
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
                                          form.getValues("limitEnabled") ? field.value : "undefined"
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
                              <FormField
                                control={form.control}
                                disabled={
                                  form.watch("limit.refill.amount") === undefined ||
                                  form.watch("limit.refill.interval") !== "monthly"
                                }
                                name="limit.refill.refillDay"
                                render={({ field }) => (
                                  <FormItem className="mt-2">
                                    <FormLabel>
                                      On which day of the month should we refill the key?
                                    </FormLabel>
                                    <FormControl>
                                      <div className="flex flex-col">
                                        <Input
                                          placeholder="Specify refill day each month"
                                          className="inline justify-end"
                                          type="number"
                                          {...field}
                                          value={
                                            form.getValues("limitEnabled")
                                              ? field.value?.toLocaleString()
                                              : undefined
                                          }
                                        />
                                      </div>
                                    </FormControl>
                                    <FormDescription>
                                      Enter the day to refill monthly.
                                    </FormDescription>
                                    <FormMessage defaultValue="Please enter a value if interval of monthly is selected" />
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
                            disabled={form.getValues("limit.refill.interval") === "daily"}
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
                        disabled={key.isLoading || !form.formState.isValid}
                        type="submit"
                        variant="primary"
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

const getDatePlusTwoMinutes = () => format(addMinutes(new Date(), 2), "yyyy-MM-dd'T'HH:mm");
