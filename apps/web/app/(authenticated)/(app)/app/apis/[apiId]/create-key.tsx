"use client";

import { CopyButton } from "@/components/dashboard/copy-button";
import { Loading } from "@/components/dashboard/loading";
import { VisibleButton } from "@/components/dashboard/visible-button";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { Button } from "@/components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "@/components/ui/card";
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
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { Textarea } from "@/components/ui/textarea";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { AlertCircle } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";

const currentTime = new Date();
const oneMinute = currentTime.setMinutes(currentTime.getMinutes() + 0.5);
const formSchema = z.object({
  bytes: z.coerce.number().positive(),
  prefix: z.string().max(8).optional(),
  ownerId: z.string().optional(),
  name: z.string().optional(),
  meta: z.string().optional(),
  // remaining: z.coerce.number().positive().optional(),
  limit: z
    .object({
      remaining: z.coerce.number().positive({ message: "Please enter a positive number" }),
      refill: z
        .object({
          interval: z.enum(["none", "daily", "monthly"]),
          amount: z.coerce
            .number()
            .int()
            .min(1, {
              message: "Please enter the number of uses per interval",
            })
            .positive(),
        })
        .optional(),
    })
    .optional(),
  expires: z.coerce.date().min(new Date(oneMinute)).optional(),
  ratelimit: z
    .object({
      type: z.enum(["consistent", "fast"]).default("fast"),
      refillInterval: z.coerce.number().positive(),
      refillRate: z.coerce.number().positive(),
      limit: z.coerce.number().positive(),
    })
    .optional(),
});

type Props = {
  apiId: string;
};

export const CreateKey: React.FC<Props> = ({ apiId }) => {
  const [expireEnabled, setExpireEnabled] = useState(false);
  const [metaEnabled, setMetaEnabled] = useState(false);
  const [limitEnabled, setLimitEnabled] = useState(false);
  const [ratelimitEnabled, setRatelimitEnabled] = useState(false);

  const { toast } = useToast();
  const router = useRouter();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      bytes: 16,
    },
  });
  const formData = form.watch();
  // if (!expireEnabled) {
  //   delete formData.expires;
  // }
  // if(!metaEnabled) {
  //   delete formData.meta;
  // }
  // if(!limitEnabled) {
  //   delete formData.limit;
  // }
  // if(!ratelimitEnabled) {
  //   delete formData.ratelimit;
  // }
  useEffect(() => {
    if (
      formData.ratelimit?.limit === undefined &&
      formData.ratelimit?.refillRate === undefined &&
      formData.ratelimit?.refillInterval === undefined
    ) {
      form.resetField("ratelimit");
    }
  }, [formData.ratelimit]);
  const key = trpc.key.create.useMutation({
    onSuccess() {
      toast({
        title: "Key Created",
        description: "Your Key has been created",
      });
      form.reset();
      router.refresh();
    },
    onError(err) {
      const errors = JSON.parse(err.message);

      if (err.data?.code === "BAD_REQUEST" && errors[0].path[0] === "ratelimit") {
        toast({
          title: "Error",
          description: "You need to include all ratelimit fields",
          variant: "alert",
        });
        return;
      }
      toast({
        title: "Error",
        description: "An error occured, please try again",
        variant: "alert",
      });
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    console.log(values);

    const _metaVal = null;
    if (values.ratelimit?.limit === undefined) {
      // delete the value to stop the server from validating it
      // as it's not required
      delete values.ratelimit;
    }
    if (!values.meta) {
      delete values.meta;
    }
    await key.mutateAsync({
      apiId,
      ...values,
      meta: values.meta ? JSON.parse(values.meta) : undefined,
      expires: values.expires?.getTime() ?? undefined,
      ownerId: values.ownerId ?? undefined,
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

  function getDatePlusTwoMinutes(): string {
    const now = new Date();
    const futureDate = new Date(now.getTime() + 2 * 60000);
    return futureDate.toISOString().slice(0, -8);
  }

  return (
    <>
      {key.data ? (
        <div className="w-full max-sm:p-4">
          <div>
            <p className="mb-4 text-xl font-bold">Your API Key</p>
            <Alert>
              <AlertCircle className="h-4 w-4" />
              <AlertTitle>This key is only shown once and can not be recovered </AlertTitle>
              <AlertDescription>
                Please pass it on to your user or store it somewhere safe.
              </AlertDescription>
            </Alert>

            <Code className="ph-no-capture my-8 flex w-full items-center justify-between gap-4 max-sm:text-xs sm:overflow-hidden">
              <pre>{showKey ? key.data.key : maskedKey}</pre>
              <div className="flex items-start justify-between gap-4 max-sm:absolute  max-sm:right-11">
                <VisibleButton isVisible={showKey} setIsVisible={setShowKey} />
                <CopyButton value={key.data.key} />
              </div>
            </Code>
          </div>

          <p className="my-2 text-center font-medium text-gray-700 ">Try verifying it:</p>
          <Code className="my-8 flex w-full items-start justify-between gap-4 overflow-hidden max-sm:text-xs ">
            <div className="max-sm:mt-10">
              <pre className="ph-no-capture">
                {showKeyInSnippet ? snippet : snippet.replace(key.data.key, maskedKey)}
              </pre>
            </div>
            <div className="max-ms:top-2 flex items-start justify-between gap-4 max-sm:absolute max-sm:right-11 ">
              <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
              <CopyButton value={snippet} />
            </div>
          </Code>
          <div className="my-4 flex justify-end space-x-4">
            <Link href={`/app/apis/${apiId}`}>
              <Button variant="secondary">Back</Button>
            </Link>
            <Button onClick={() => key.reset()}>Create another key</Button>
          </div>
        </div>
      ) : (
        <>
          <div>
            <div>
              <Form {...form}>
                <form className="flex flex-col" onSubmit={form.handleSubmit(onSubmit)}>
                  <h2 className="mb-2 text-2xl">Create a new Key</h2>
                  <div className="flex-col gap-y-4">
                    <Card className="p-2">
                      <CardHeader>
                        <CardTitle>Key Details</CardTitle>
                        <CardDescription />
                      </CardHeader>
                      <CardContent className="flex justify-between item-center gap-4">
                        <div className="flex w-1/4 ">
                          <FormField
                            control={form.control}
                            name="prefix"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>Prefix</FormLabel>
                                <FormControl>
                                  <Input {...field} />
                                </FormControl>
                                <FormDescription>
                                  Using a prefix can make it easier for your users to distinguish
                                  between apis. Don't add a trailing underscore, we'll do that
                                  automatically.
                                </FormDescription>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>
                        <div className="flex-col w-1/4">
                          <FormField
                            control={form.control}
                            name="bytes"
                            rules={{ required: true }}
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
                        </div>
                        <div className="flex-col w-1/4">
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
                        </div>
                        <div className="flex-col w-1/4">
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
                                  To make it easier to identify a particular key, you can provide a
                                  name.
                                </FormDescription>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </div>
                      </CardContent>
                    </Card>
                    <Accordion type="multiple" className="w-full">
                      <AccordionItem value="advanced">
                        <AccordionTrigger dir="">Advanced</AccordionTrigger>
                        <AccordionContent className="w-full">
                          <div className="flex gap-4 mb-4">
                            <div className="lg:w-1/3">
                              <Card className="h-full">
                                <CardHeader>
                                  <CardTitle>Ratelimit</CardTitle>
                                  <CardDescription>
                                    How frequently this key can be used.
                                  </CardDescription>
                                </CardHeader>
                                <div className="flex items-center gap-4 pt-6 pl-6">
                                  <Switch
                                    id="enabled"
                                    checked={ratelimitEnabled}
                                    onCheckedChange={setRatelimitEnabled}
                                  />
                                  <Label htmlFor="enabled">
                                    {ratelimitEnabled ? "Enabled" : "Disabled"}
                                  </Label>
                                </div>
                                <CardContent className="w-full justify-between item-center">
                                  <div className="">
                                    <FormField
                                      control={form.control}
                                      name="ratelimit.limit"
                                      render={({ field }) => (
                                        <FormItem className="w-full mt-2">
                                          <FormLabel>Limit</FormLabel>
                                          <FormControl>
                                            <Input
                                              disabled={!ratelimitEnabled}
                                              placeholder="10"
                                              type="number"
                                              {...field}
                                              onBlur={(e) => {
                                                if (e.target.value === "") {
                                                  //don't trigger validation if the field is empty
                                                  return;
                                                }
                                              }}
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
                                        <FormItem className="w-full mt-4">
                                          <FormLabel>Refill Rate</FormLabel>
                                          <FormControl>
                                            <Input
                                              disabled={!ratelimitEnabled}
                                              placeholder="5"
                                              type="number"
                                              {...field}
                                              onBlur={(e) => {
                                                if (e.target.value === "") {
                                                  return;
                                                }
                                              }}
                                            />
                                          </FormControl>

                                          <FormMessage />
                                        </FormItem>
                                      )}
                                    />
                                    <FormField
                                      control={form.control}
                                      name="ratelimit.refillInterval"
                                      render={({ field }) => (
                                        <FormItem className="w-full mt-6">
                                          <FormLabel>Refill Interval (milliseconds)</FormLabel>
                                          <FormControl>
                                            <Input
                                              disabled={!ratelimitEnabled}
                                              placeholder="1000"
                                              type="number"
                                              {...field}
                                              onBlur={(e) => {
                                                if (e.target.value === "") {
                                                  return;
                                                }
                                              }}
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
                                </CardContent>
                              </Card>
                            </div>

                            <div className="lg:w-1/3">
                              <Card className="h-full">
                                <CardHeader>
                                  <CardTitle>Remaining Uses</CardTitle>
                                  <CardDescription>
                                    How many times this key can be used before it gets disabled
                                    automatically.
                                  </CardDescription>
                                </CardHeader>
                                <div className="flex items-center gap-4 pt-6 pl-6">
                                  <Switch
                                    id="enableRemaining"
                                    checked={limitEnabled}
                                    onCheckedChange={setLimitEnabled}
                                  />
                                  <Label htmlFor="enableRemaining">
                                    {limitEnabled ? "Enabled" : "Disabled"}
                                  </Label>
                                </div>
                                <CardContent className="justify-between item-center">
                                  <div>
                                    <FormField
                                      control={form.control}
                                      name="limit.remaining"
                                      render={({ field }) => (
                                        <FormItem>
                                          <FormLabel>Number of uses</FormLabel>
                                          <FormControl>
                                            <Input
                                              disabled={!limitEnabled}
                                              placeholder="100"
                                              className="w-full"
                                              type="number"
                                              {...field}
                                              onBlur={(e) => {
                                                if (e.target.value === "") {
                                                  return;
                                                }
                                              }}
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
                                            disabled={!limitEnabled}
                                            onValueChange={field.onChange}
                                            defaultValue=""
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
                                      name="limit.refill.amount"
                                      render={({ field }) => (
                                        <FormItem className="mt-4">
                                          <FormLabel>Number of uses per interval</FormLabel>
                                          <FormControl>
                                            <Input
                                              disabled={!limitEnabled}
                                              placeholder="100"
                                              className="w-full"
                                              type="number"
                                              {...field}
                                              onBlur={(e) => {
                                                if (e.target.value === "") {
                                                  return;
                                                }
                                              }}
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
                              </Card>
                            </div>
                            <div className="lg:w-1/3">
                              <Card className="h-full">
                                <CardHeader>
                                  <CardTitle>Expiration</CardTitle>
                                  <CardDescription>
                                    Automatically revoke this key after a certain date.
                                  </CardDescription>
                                </CardHeader>
                                <div className="flex items-center gap-4 pl-6 pt-6">
                                  <Switch
                                    id="enableExpiration"
                                    checked={expireEnabled}
                                    onCheckedChange={setExpireEnabled}
                                  />
                                  <Label htmlFor="enableExpiration">
                                    {expireEnabled ? "Enabled" : "Disabled"}
                                  </Label>
                                </div>

                                <CardContent className="justify-between item-center h-[328px]">
                                  <div
                                    className={cn("flex flex-col gap-2 w-full", {
                                      "opacity-50": !expireEnabled,
                                    })}
                                  >
                                    <FormField
                                      control={form.control}
                                      name="expires"
                                      render={({ field }) => (
                                        <FormItem>
                                          <FormLabel>Expiry Date</FormLabel>
                                          <FormControl>
                                            <Input
                                              disabled={!expireEnabled}
                                              min={getDatePlusTwoMinutes()}
                                              type="datetime-local"
                                              {...field}
                                              defaultValue={getDatePlusTwoMinutes()}
                                              value={field.value?.toLocaleString()}
                                            />
                                          </FormControl>
                                          <FormDescription>
                                            This api key will automatically be revoked after the
                                            given date.
                                          </FormDescription>
                                          <FormMessage />
                                        </FormItem>
                                      )}
                                    />
                                  </div>
                                </CardContent>
                              </Card>
                            </div>
                          </div>
                          <div className="flex flex-col w-full">
                            <Card>
                              <CardHeader>
                                <CardTitle>Metadata</CardTitle>
                                <CardDescription>
                                  Store json, or any other data you want to associate with this key.
                                  Whenever you verify this key, we'll return the metadata to you.
                                </CardDescription>
                              </CardHeader>
                              <div className="flex items-center gap-4 pl-6 pt-6">
                                <Switch
                                  id="enableMetadata"
                                  checked={metaEnabled}
                                  onCheckedChange={setMetaEnabled}
                                />
                                <Label htmlFor="enableMetadata">
                                  {!metaEnabled ? "Enabled" : "Disabled"}
                                </Label>
                              </div>
                              <CardContent className="justify-between item-center">
                                <div
                                  className={cn("flex flex-col gap-2 w-full", {
                                    "opacity-50": !metaEnabled,
                                  })}
                                >
                                  <FormField
                                    control={form.control}
                                    name="meta"
                                    render={({ field }) => (
                                      <FormItem>
                                        <FormControl>
                                          <Textarea
                                            disabled={!metaEnabled}
                                            className="m-4 mx-auto rounded-md border shadow-sm"
                                            rows={7}
                                            placeholder={`{"stripeCustomerId" : "cus_9s6XKzkNRiz8i3"}`}
                                            {...field}
                                          />
                                        </FormControl>
                                        <FormDescription>
                                          Enter custom metadata as a JSON object.
                                        </FormDescription>
                                        <FormMessage />
                                        <Button
                                          disabled={!metaEnabled}
                                          variant="secondary"
                                          type="button"
                                          onClick={(_e) => {
                                            try {
                                              if (field.value) {
                                                const parsed = JSON.parse(field.value);
                                                field.onChange(JSON.stringify(parsed, null, 2));
                                              }
                                            } catch (_e) {
                                              form.setError("meta", {
                                                type: "manual",
                                                message: "Invalid JSON",
                                              });
                                            }
                                          }}
                                        >
                                          Format Json
                                        </Button>
                                      </FormItem>
                                    )}
                                  />
                                </div>
                              </CardContent>
                            </Card>
                          </div>
                        </AccordionContent>
                      </AccordionItem>
                    </Accordion>
                  </div>
                  <div className="mt-8 flex justify-end">
                    <Button disabled={!form.formState.isValid || key.isLoading} type="submit">
                      {key.isLoading ? <Loading /> : "Create"}
                    </Button>
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
