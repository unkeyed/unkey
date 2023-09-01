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
import { Textarea } from "@/components/ui/textarea";
import { useToast } from "@/components/ui/use-toast";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { AlertCircle } from "lucide-react";
import Link from "next/link";
import { useRouter } from "next/navigation";
import { useEffect, useState } from "react";
import { useForm } from "react-hook-form";
import { z } from "zod";
const formSchema = z.object({
  bytes: z.coerce.number().positive(),
  prefix: z.string().max(8).optional(),
  ownerId: z.string().optional(),
  meta: z.record(z.unknown()).optional(),
  remaining: z.coerce.number().positive().optional(),
  expires: z
    .string()
    .transform((s) => new Date(s).getTime())
    .optional(),
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
  const { toast } = useToast();
  const router = useRouter();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      prefix: "key",
      bytes: 16,
    },
  });
  const formData = form.watch();
  useEffect(() => {
    if (formData.ratelimit?.limit === undefined && formData.ratelimit?.refillRate === undefined && formData.ratelimit?.refillInterval === undefined) {
      form.resetField("ratelimit")
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
      expires: values.expires ?? undefined,
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
  return (
    <>
      {key.data ? (
        <div className="w-full">
          <div>
            <p className="mb-4 text-xl font-bold">Your API Key</p>
            <Alert>
              <AlertCircle className="w-4 h-4" />
              <AlertTitle>This key is only shown once and can not be recovered </AlertTitle>
              <AlertDescription>
                Please pass it on to your user or store it somewhere safe.
              </AlertDescription>
            </Alert>

            <Code className="flex items-center justify-between w-full gap-4 my-8 ">
              <pre>{showKey ? key.data.key : maskedKey}</pre>
              <div className="flex items-start justify-between gap-4">
                <VisibleButton isVisible={showKey} setIsVisible={setShowKey} />
                <CopyButton value={key.data.key} />
              </div>
            </Code>
          </div>

          <p className="my-2 font-medium text-center text-gray-700 ">Try verifying it:</p>
          <Code className="flex items-start justify-between w-full gap-4 my-8 ">
            {showKeyInSnippet ? snippet : snippet.replace(key.data.key, maskedKey)}
            <div className="flex items-start justify-between gap-4">
              <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
              <CopyButton value={snippet} />
            </div>
          </Code>
          <div className="flex justify-end my-4 space-x-4">
            <Link href={`/app/${apiId}`}>
              <Button variant="secondary">Back</Button>
            </Link>
            <Button onClick={() => key.reset()}>Create another key</Button>
          </div>
        </div>
      ) : (
        <>
          <div>
            <div className="w-full overflow-scroll">
              <h2 className="mb-2 text-2xl">Create a new Key</h2>
              <Form {...form}>
                <form className="max-w-6xl mx-auto" onSubmit={form.handleSubmit(onSubmit)}>
                  <div className="flex justify-evenly gap-4">
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
                            Using a prefix can make it easier for your users to distinguish between
                            apis
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
                  </div>
                  <h3 className="my-4 text-xl">Advanced</h3>
                  <div className="grid grid-cols-1 gap-4 my-4 md:grid-cols-2 ">
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
                                  />
                                </FormControl>
                                <FormDescription>
                                  The maximum number of requests possible during a burst.
                                </FormDescription>
                                <FormMessage />
                              </FormItem>
                            )}
                          />

                          <div className="flex items-center gap-4 mt-8">
                            <FormField
                              control={form.control}
                              name="ratelimit.refillRate"
                              render={({ field }) => (
                                <FormItem className="w-full">
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
                                <FormItem className="w-full">
                                  <FormLabel>Refill Interval (milliseconds)</FormLabel>
                                  <FormControl>
                                    <Input placeholder="1000" type="number" {...field} />
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
                            name="remaining"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>Number of uses</FormLabel>
                                <FormControl>
                                  <Input
                                    placeholder="100"
                                    className="w-full"
                                    type="number"
                                    {...field}
                                  />
                                </FormControl>
                                <FormDescription>
                                  Enter the remaining amount of uses for this key.
                                </FormDescription>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </AccordionContent>
                      </AccordionItem>
                    </Accordion>
                    <Accordion type="multiple" className="w-full">
                      <AccordionItem value="metadata" className="w-full">
                        <AccordionTrigger dir="">Add Custom Metadata</AccordionTrigger>
                        <AccordionContent className="flex w-full">
                          <FormField
                            control={form.control}
                            name="meta"
                            render={({ field }) => (
                              <FormItem>
                                <FormControl>
                                  <Textarea
                                    className="border rounded-md shadow-sm m-4 max-w-full w-96"
                                    placeholder={`{"stripeCustomerId" : "cus_9s6XKzkNRiz8i3"}`}
                                    onBlur={(e) => {
                                      try {
                                        const value = JSON.parse(e.target.value);
                                        const prettier = JSON.stringify(value, null, 2);
                                        e.target.value = prettier;
                                        field.onChange(JSON.parse(e.target.value));
                                        form.clearErrors("meta");
                                      } catch (_e) { }
                                    }}
                                    onChange={(e) => {
                                      try {
                                        field.onChange(JSON.parse(e.target.value));
                                        form.clearErrors("meta");
                                      } catch (_e) {
                                        form.setError("meta", {
                                          type: "manual",
                                          message: "Invalid JSON",
                                        });
                                      }
                                    }}
                                  />
                                </FormControl>
                                <FormDescription>
                                  Enter custom metadata as a JSON object.
                                </FormDescription>
                                <FormMessage />
                              </FormItem>
                            )}
                          />
                        </AccordionContent>
                      </AccordionItem>
                    </Accordion>
                    <Accordion type="multiple" className="w-full">
                      <AccordionItem value="expiration">
                        <AccordionTrigger dir="">Add Expiration </AccordionTrigger>
                        <AccordionContent className="flex w-full ">
                          <FormField
                            control={form.control}
                            name="expires"
                            render={({ field }) => (
                              <FormItem>
                                <FormLabel>Expiry Date</FormLabel>
                                <FormControl>
                                  <Input type="date" {...field} />
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
                  <div className="flex justify-end mt-8">
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
