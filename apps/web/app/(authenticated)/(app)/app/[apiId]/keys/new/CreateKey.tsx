"use client";
import { useRouter } from "next/navigation";
import { useToast } from "@/components/ui/use-toast";
import {
  Form,
  FormControl,
  FormDescription,
  FormField,
  FormItem,
  FormLabel,
  FormMessage,
} from "@/components/Form";
import { Loading } from "@/components/loading";
import { Button } from "@/components/ui/button";
import { Switch } from "@/components/ui/switch";
import {
  Accordion,
  AccordionContent,
  AccordionItem,
  AccordionTrigger,
} from "@/components/ui/accordion";
import { Input } from "@/components/ui/input";
import { CopyButton } from "@/components/CopyButton";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { useState } from "react";
import { VisibleButton } from "@/components/VisibleButton";

const formSchema = z.object({
  bytes: z.coerce.number().positive(),
  prefix: z.string().max(8).optional(),
  ownerId: z.string().optional(),
  meta: z.record(z.unknown()).optional(),
  expiresEnabled: z.boolean().default(false),
  expires: z
    .string()
    .transform((s) => new Date(s).getTime())
    .optional(),
  rateLimitEnabled: z.boolean().default(false),
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
    defaultValues: {
      prefix: "key",
      bytes: 16,
    },
  });
  const key = trpc.key.create.useMutation({
    onSuccess() {
      toast({
        title: "Key Created",
        description: "Your Key has been created",
      });
      form.reset();
    },
    onError(err) {
      const errors = JSON.parse(err.message);
      if (err.data?.code === "BAD_REQUEST" && errors[0].path[0] === "ratelimit") {
        toast({
          title: "Error",
          description: "You need to include all ratelimit fields",
          variant: "destructive",
        });
        return;
      }
      toast({
        title: "Error",
        description: "An error occured, please try again",
        variant: "destructive",
      });
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (!values.rateLimitEnabled || values.ratelimit === undefined) {
      // delete the value to stop the server from validating it
      // as it's not required
      values.ratelimit = undefined;
    }

    await key.mutateAsync({
      apiId,
      ...values,
      expires: values.expires ?? undefined,
      ownerId: values.ownerId ?? undefined,
    });
  }

  const snippet = `curl -XPOST 'https://api.unkey.dev/v1/keys/verify' \\
  -H 'Content-Type: application/json' \\
  -d '{
    "key": "${key.data?.key}"
  }'
  `;

  const maskedKey = `unkey_${"*".repeat(key.data?.key.split("_").at(1)?.length ?? 0)}`;
  const [showKey, setShowKey] = useState(false);
  const [showKeyInSnippet, setShowKeyInSnippet] = useState(false);
  return (
    <>
      {key.data ? (
        <div className="w-full">
          <div>
            <p className="mb-4 text-xl font-bold">Your API Key</p>
            <div
              className="flex justify-center max-w-3xl p-4 mx-auto mb-4 text-sm text-yellow-800 border border-yellow-300 rounded-lg bg-yellow-50 dark:bg-gray-800 dark:text-yellow-300 dark:border-yellow-800"
              role="alert"
            >
              <svg
                aria-hidden="true"
                className="flex-shrink-0 inline w-5 h-5 mr-3"
                fill="currentColor"
                viewBox="0 0 20 20"
                xmlns="http://www.w3.org/2000/svg"
              >
                <path
                  fill-rule="evenodd"
                  d="M18 10a8 8 0 11-16 0 8 8 0 0116 0zm-7-4a1 1 0 11-2 0 1 1 0 012 0zM9 9a1 1 0 000 2v3a1 1 0 001 1h1a1 1 0 100-2v-3a1 1 0 00-1-1H9z"
                  clip-rule="evenodd"
                />
              </svg>
              <span className="sr-only">Info</span>
              <div>
                <span className="font-medium">Warning!</span> This key is only shown once and can
                not be recovered. Please store it somewhere safe.
              </div>
            </div>

            <div className="flex items-center justify-between max-w-6xl gap-4 px-2 py-1 mx-auto mt-4 border rounded lg:p-4 border-white/10 bg-zinc-100 dark:bg-zinc-900">
              <pre className="font-mono">{showKey ? key.data.key : maskedKey}</pre>
              <div className="flex items-start justify-between gap-4">
                <VisibleButton isVisible={showKey} setIsVisible={setShowKey} />
                <CopyButton value={key.data.key} />
              </div>
            </div>
          </div>

          <p className="mt-2 mb-2 text-lg font-medium text-center text-zinc-700 ">
            Try verifying it:
          </p>
          <div className="flex items-start justify-between max-w-6xl gap-4 px-2 py-1 mx-auto border rounded lg:p-4 border-white/10 bg-zinc-100 dark:bg-zinc-900">
            <pre className="font-mono">
              {showKeyInSnippet ? snippet : snippet.replace(key.data.key, maskedKey)}
            </pre>
            <div className="flex items-start justify-between gap-4">
              <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
              <CopyButton value={snippet} />
            </div>
          </div>
          <div className="flex justify-around my-4 space-x-4">
            <Button className="w-1/4" onClick={() => key.reset()}>
              Create another key
            </Button>
            <Button
              variant="outline"
              className="w-1/4 border-zinc-700"
              onClick={() => router.push(`/app/${apiId}`)}
            >
              {" "}
              Done{" "}
            </Button>
          </div>
        </div>
      ) : (
        <>
          <div>
            <div className="w-full overflow-scroll">
              <h2 className="mb-2 text-2xl text-center">Create a new Key</h2>
              <Form {...form}>
                <form className="max-w-6xl mx-auto" onSubmit={form.handleSubmit(onSubmit)}>
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

                  <div className="flex justify-around my-4 space-x-4">
                    <FormField
                      control={form.control}
                      name="expiresEnabled"
                      render={({ field }) => {
                        return (
                          <FormItem>
                            <FormControl>
                              <div className="flex items-center space-x-4">
                                <Switch
                                  checked={field.value}
                                  defaultValue={"false"}
                                  onCheckedChange={(value) => {
                                    if (value === false) {
                                      form.resetField("expires");
                                      form.clearErrors("expires");
                                    }
                                    field.onChange(value);
                                  }}
                                />
                                <FormLabel>Enable Expiration</FormLabel>
                              </div>
                            </FormControl>
                          </FormItem>
                        );
                      }}
                    />

                    <FormField
                      control={form.control}
                      name="rateLimitEnabled"
                      render={({ field }) => {
                        return (
                          <FormItem>
                            <FormControl>
                              <div className="flex items-center space-x-4">
                                <Switch
                                  checked={field.value}
                                  defaultValue={"false"}
                                  onCheckedChange={(value) => {
                                    if (value === false) {
                                      form.resetField("ratelimit");
                                    }
                                    field.onChange(value);
                                  }}
                                />
                                <FormLabel>Enable Ratelimiting</FormLabel>
                              </div>
                            </FormControl>
                          </FormItem>
                        );
                      }}
                    />
                  </div>
                  {form.watch("expiresEnabled") && (
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
                  )}
                  {form.watch("rateLimitEnabled") && (
                    <>
                      <FormField
                        control={form.control}
                        name="ratelimit.limit"
                        render={({ field }) => (
                          <FormItem className="w-full">
                            <FormLabel>Limit</FormLabel>
                            <FormControl>
                              <Input autoFocus={true} type="number" {...field} />
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
                                <Input type="number" {...field} />
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
                                <Input type="number" {...field} />
                              </FormControl>

                              <FormMessage />
                            </FormItem>
                          )}
                        />
                      </div>
                      <FormDescription>
                        How many requests may be performed in a given interval
                      </FormDescription>
                    </>
                  )}
                  <Accordion type="multiple" className="w-full">
                    <AccordionItem disabled value="item-3">
                      <AccordionTrigger dir="">Add Policies (soon)</AccordionTrigger>
                      <AccordionContent>TODO: andreas</AccordionContent>
                    </AccordionItem>
                  </Accordion>
                  <div className="justify-end mt-8">
                    <Button disabled={!form.formState.isValid} type="submit">
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
