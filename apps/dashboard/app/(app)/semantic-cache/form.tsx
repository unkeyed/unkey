"use client";
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
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { PostHogEvent } from "@/providers/PostHogProvider";
import { zodResolver } from "@hookform/resolvers/zod";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { generateSemanticCacheDefaultName } from "./new/util/generate-semantic-cache-default-name";
const formSchema = z.object({
  subdomain: z.string().regex(/^[a-zA-Z0-9-]+$/),
});

export const CreateLLMGatewayForm: React.FC = () => {
  const router = useRouter();
  const defaultName = generateSemanticCacheDefaultName();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    defaultValues: {
      subdomain: defaultName,
    },
  });

  const create = trpc.llmGateway.create.useMutation({
    onSuccess(res) {
      toast.success("Gateway Created", {
        description: "Your Gateway has been created",
        duration: 10_000,
      });
      PostHogEvent({
        name: "semantic_cache_gateway_created",
        properties: { id: res.id },
      });
      router.push(`/semantic-cache/${res.id}/logs`);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    const gatewayValues = {
      subdomain: values.subdomain,
    };

    create.mutate(gatewayValues);
  }

  return (
    <div>
      <div className="flex items-start justify-between gap-16 mt-8">
        <div className="space-y-2">
          <h4 className="text-lg font-medium">What is semantic caching?</h4>
          <p className="text-sm text-content-subtle">
            Faster, cheaper LLM API calls through re-using semantically similar previous responses.
          </p>
          <ol className="ml-2 space-y-1 text-sm list-decimal list-outside text-content-subtle">
            <li>You switch out the baseUrl in your requests to OpenAI with your gateway URL</li>
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
                              .llm.unkey.io
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
};
