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
import Link from "next/link";
import { Alert, AlertDescription, AlertTitle } from "@/components/ui/alert";
import { AlertCircle } from "lucide-react";
import { Code } from "@/components/ui/code";
import { Card, CardContent } from "@/components/ui/card";

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
  const _router = useRouter();
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
      delete values.ratelimit
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

          <p className="my-2 font-medium text-center text-zinc-700 ">Try verifying it:</p>
          <Code className="flex items-start justify-between w-full gap-4 my-8 ">
            {showKeyInSnippet ? snippet : snippet.replace(key.data.key, maskedKey)}
            <div className="flex items-start justify-between gap-4">
              <VisibleButton isVisible={showKeyInSnippet} setIsVisible={setShowKeyInSnippet} />
              <CopyButton value={snippet} />
            </div>
          </Code>
          <div className="flex justify-end my-4 space-x-4">
            <Link href={`/app/${apiId}`}>
              <Button variant="outline">Back</Button>
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
