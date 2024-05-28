"use client";
import { StackedColumnChart } from "@/components/dashboard/charts";
import { Accordion } from "@/components/ui/accordion";
import { Button } from "@/components/ui/button";
import { Card, CardContent } from "@/components/ui/card";
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
import { Separator } from "@/components/ui/separator";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { sub } from "date-fns";
import { DatabaseZap } from "lucide-react";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  subdomain: z.string().regex(/^[a-zA-Z0-9-]+$/),
  apiKey: z.string(),
});

export default function EnableSemanticCacheForm() {
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
  });

  const create = trpc.gateway.create.useMutation({
    onSuccess() {
      toast.success("Gateway Created", {
        description: "Your Gateway has been created",
        duration: 10_000,
      });
    },
    onError(err) {
      toast.error("An error occured", {
        description: err.message,
      });
    },
  });

  // test

  async function onSubmit(values: z.infer<typeof formSchema>) {
    console.info("submit");
    const gatewayValues = {
      subdomain: values.subdomain,
      origin: "https://unkey.dev",
      headerRewrites: [
        {
          name: "OPENAI_API_KEY",
          value: values.apiKey,
          show: false,
        },
      ],
    };

    create.mutate(gatewayValues);

    console.info("created");
  }

  return (
    <div>
      <div className="flex items-start justify-between gap-16 mt-8">
        <div className="space-y-2">
          <div className="inline-flex items-center justify-center p-4 border rounded-full bg-primary/5">
            <DatabaseZap className="w-6 h-6 text-primary" />
          </div>
          <h4 className="text-lg font-medium">What is semantic caching?</h4>
          <p className="text-sm text-content-subtle">
            Faster, cheaper LLM API calls through re-using semantically similar previous responses.
          </p>
          <ol className="ml-2 space-y-1 text-sm list-decimal list-outside text-content-subtle">
            <li>You switch out the baseUrl in your requests to OpenAI with your gateway URL</li>
            <li>
              You add your OpenAI API key here, and replace the API key in your requests with your
              gateway key
            </li>
            <li>Unkey will automatically start caching your responses</li>
            <li>Monitor and track your cache usage here</li>
          </ol>
        </div>

        <div className="w-full">
          <Card>
            <CardContent>
              <Form {...form}>
                <form className="flex flex-col gap-8" onSubmit={form.handleSubmit(onSubmit)}>
                  <FormField
                    control={form.control}
                    name="subdomain"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>Domain</FormLabel>
                        <FormControl>
                          <div className="flex mt-2 border rounded-md shadow-sm focus-within:border-primary">
                            <span className="inline-flex items-center px-3 text-content bg-background-subtle rounded-l-md sm:text-sm ">
                              https://
                            </span>
                            <input
                              className="flex-1 block w-full h-8 min-w-0 px-3 py-2 text-sm bg-transparent placeholder:text-content-subtle focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50 "
                              // className="flex w-full h-8 px-3 py-2 text-sm bg-transparent border rounded-md border-border focus:border-primary file:border-0 file:bg-transparent file:text-sm file:font-medium placeholder:text-content-subtle focus-visible:outline-none disabled:cursor-not-allowed disabled:opacity-50"
                              {...field}
                            />
                            <span className="inline-flex items-center px-3 text-content bg-background-subtle rounded-r-md sm:text-sm">
                              .llmcache.unkey.dev
                            </span>
                          </div>
                        </FormControl>
                        <FormDescription>
                          The domain where your semantic cache will be available
                        </FormDescription>
                        <FormMessage />
                      </FormItem>
                    )}
                  />
                  <FormField
                    control={form.control}
                    name="apiKey"
                    render={({ field }) => (
                      <FormItem>
                        <FormLabel>OpenAI API Key</FormLabel>
                        <FormControl>
                          <Input type="password" {...field} />
                        </FormControl>
                        <FormMessage />
                      </FormItem>
                    )}
                  />

                  <Separator />

                  <div className="w-full">
                    <Button className="w-full" type="submit" variant="primary">
                      Deploy
                    </Button>
                  </div>
                </form>
              </Form>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}
