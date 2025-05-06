"use client";
import { Loading } from "@/components/dashboard/loading";
import {
  Card,
  CardContent,
  CardDescription,
  CardFooter,
  CardHeader,
  CardTitle,
} from "@/components/ui/card";
import { Form, FormControl, FormField, FormItem, FormMessage } from "@/components/ui/form";
import { Label } from "@/components/ui/label";
import { Textarea } from "@/components/ui/textarea";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  keyId: z.string(),
  metadata: z.string(),
});
type Props = {
  apiKey: {
    id: string;
    meta: string | null;
  };
};

export const UpdateKeyMetadata: React.FC<Props> = ({ apiKey }) => {
  const router = useRouter();

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      keyId: apiKey.id,
      metadata: apiKey.meta ?? "",
    },
  });

  const rows = Math.max(3, form.getValues("metadata").split("\n").length);

  const updateMetadata = trpc.key.update.metadata.useMutation({
    onSuccess() {
      toast.success("Your metadata has been updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    updateMetadata.mutate({
      keyId: values.keyId,
      metadata: {
        enabled: Boolean(values.metadata),
        data: values.metadata,
      },
    });
  }
  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardHeader>
            <CardTitle>Metadata</CardTitle>
            <CardDescription>
              Store json, or any other data you want to associate with this key. Whenever you verify
              this key, we'll return the metadata to you.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex justify-between item-center">
            <div className="flex flex-col w-full space-y-2">
              <Label htmlFor="metadata">Metadata</Label>
              <FormField
                control={form.control}
                name="metadata"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <Textarea
                        {...field}
                        rows={rows}
                        value={form.getValues("metadata")}
                        className="w-full"
                        defaultValue={apiKey.meta ?? ""}
                        autoComplete="off"
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </CardContent>
          <CardFooter className="items-center justify-end gap-4">
            <Button
              type="button"
              onClick={() => {
                try {
                  const parsed = JSON.parse(form.getValues("metadata"));
                  form.setValue("metadata", JSON.stringify(parsed, null, 2));
                } catch (e) {
                  toast.error((e as Error).message);
                }
              }}
            >
              Format Json
            </Button>
            <Button disabled={updateMetadata.isLoading || !form.formState.isValid} type="submit">
              {updateMetadata.isLoading ? <Loading /> : "Save"}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
