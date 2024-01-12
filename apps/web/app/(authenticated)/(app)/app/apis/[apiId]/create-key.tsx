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

const currentTime = new Date();
const oneMinute = currentTime.setMinutes(currentTime.getMinutes() + 0.5);
const formSchema = z.object({
  bytes: z.coerce.number().positive({ message: "Please enter a positive number" }),
  prefix: z
    .string()
    .max(8, { message: "Please limit the prefix to under 8 characters." })
    .optional(),
  ownerId: z.string().optional(),
  name: z.string().optional(),
  meta: z.string().optional(),
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

  useEffect(() => {
    if (formData.limit?.remaining === undefined) {
      form.resetField("limit");
    }
  }, [formData.limit]);
  useEffect(() => {
    if (formData.limit?.refill?.interval === "none") {
      form.resetField("limit.refill.interval");
      form.resetField("limit.refill.amount");
    }
  }, [formData.limit?.refill]);
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
      toast("Key Created", {
        description: "Your Key has been created",
      });
      form.reset();
      router.refresh();
    },
    onError(err) {
      const errors = JSON.parse(err.message);

      if (err.data?.code === "BAD_REQUEST" && errors[0].path[0] === "ratelimit") {
        toast.error("You need to include all ratelimit fields");
        return;
      }
      toast.error("An error occured, please try again");
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    const _metaVal = null;
    if (values.ratelimit?.limit === undefined) {
      // delete the value to stop the server from validating it
      // as it's not required
      delete values.ratelimit;
    }
    if (!values.meta) {
      delete values.meta;
    }
    if (
      values.limit?.refill &&
      values.limit?.refill?.interval !== "none" &&
      values.limit?.remaining === undefined
    ) {
      form.setError("limit.remaining", {
        type: "manual",
        message: "Please enter a value if interval is selected",
      });
      return;
    }

    if (
      values.limit &&
      values.limit?.refill?.interval !== "daily" &&
      values.limit?.refill?.interval !== "monthly"
    ) {
      delete values.limit.refill;
    }
    if (values.limit?.remaining === undefined) {
      delete values.limit;
    }

    await key.mutateAsync({
      apiId,
      ...values,
      meta: values.meta ? JSON.parse(values.meta) : undefined,
      expires: values.expires?.getTime() ?? undefined,
      ownerId: values.ownerId ?? undefined,
      remaining: values.limit?.remaining ?? undefined,
      refill:
        values.limit?.refill && values.limit.refill.interval !== "none"
          ? {
              interval: values.limit.refill.interval,
              amount: values.limit.refill.amount,
            }
          : undefined,
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
            <div className="w-full overflow-scroll">
              <Form {...form}>
                <form className="mx-auto max-w-6xl" onSubmit={form.handleSubmit(onSubmit)}>
                  <h2 className="mb-2 text-2xl">Create a New Key</h2>
                  <div className="flex flex-col justify-evenly gap-4 md:flex-row">
                    <FormField
                      control={form.control}
                      name="prefix"
                      render={({ field }) => (
                        <FormItem className="w-full md:w-1/4">
                          <FormLabel>Prefix</FormLabel>
                          <FormControl>
                            <Input {...field} />
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
                      rules={{ required: true }}
                      render={({ field }) => (
                        <FormItem className="w-full md:w-1/4">
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
                        <FormItem className="w-full md:w-1/4">
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
                        <FormItem className="w-full md:w-1/4">
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
                  <h3 className="my-4 text-xl">Advanced</h3>
                  <div className="flex flex-col md:flex-row ">
                    <div className="w-full px-4">
                      <Accordion type="multiple" className="w-full">
                        <AccordionItem value="ratelimit">
                          <AccordionTrigger dir="">Add Ratelimiting</AccordionTrigger>
                          <AccordionContent>
                            <FormField
                              control={form.control}
                              name="ratelimit.limit"
                              render={({ field }) => (
                                <FormItem className="w-full">
                                  <FormLabel>Limit</FormLabel>
                                  <FormControl>
                                    <Input
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
                            <div className="mt-8 flex items-center gap-4">
                              <FormField
                                control={form.control}
                                name="ratelimit.refillRate"
                                render={({ field }) => (
                                  <FormItem className="w-full">
                                    <FormLabel>Refill Rate</FormLabel>
                                    <FormControl>
                                      <Input
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
                                  <FormItem className="w-full">
                                    <FormLabel>Refill Interval (milliseconds)</FormLabel>
                                    <FormControl>
                                      <Input
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
                            </div>
                            <FormDescription>
                              How many requests may be performed in a given interval
                            </FormDescription>
                          </AccordionContent>
                        </AccordionItem>
                      </Accordion>
                      <Accordion type="multiple" className="w-full">
                        <AccordionItem value="limit">
                          <AccordionTrigger dir="">Limit Usage</AccordionTrigger>
                          <AccordionContent>
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
                                <FormItem className="mt-4">
                                  <FormLabel>Refill Rate</FormLabel>
                                  <Select onValueChange={field.onChange} defaultValue="">
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
                          </AccordionContent>
                        </AccordionItem>
                      </Accordion>
                    </div>
                    <div className="w-full px-4">
                      <Accordion type="multiple" className="w-full">
                        <AccordionItem value="metadata">
                          <AccordionTrigger dir="">Custom Metadata</AccordionTrigger>
                          <AccordionContent>
                            <FormField
                              control={form.control}
                              name="meta"
                              render={({ field }) => (
                                <FormItem>
                                  <FormControl>
                                    <Textarea
                                      className="m-4 mx-auto w-full rounded-md border shadow-sm"
                                      rows={3}
                                      placeholder={`{"stripeCustomerId" : "cus_9s6XKzkNRiz8i3"}`}
                                      {...field}
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
                          </AccordionContent>
                        </AccordionItem>
                      </Accordion>
                      <Accordion type="multiple" className="w-full">
                        <AccordionItem value="expiration-field  ">
                          <AccordionTrigger dir="">Add Expiration</AccordionTrigger>
                          <AccordionContent>
                            <FormField
                              control={form.control}
                              name="expires"
                              render={({ field }) => (
                                <FormItem>
                                  <FormLabel>Expiry Date</FormLabel>
                                  <FormControl>
                                    <Input
                                      min={getDatePlusTwoMinutes()}
                                      type="datetime-local"
                                      {...field}
                                      defaultValue={getDatePlusTwoMinutes()}
                                      value={field.value?.toLocaleString()}
                                    />
                                  </FormControl>
                                  <FormDescription>
                                    This api key will automatically be revoked after the given date.
                                  </FormDescription>
                                  <FormMessage />
                                </FormItem>
                              )}
                            />
                          </AccordionContent>
                        </AccordionItem>
                      </Accordion>
                    </div>
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
