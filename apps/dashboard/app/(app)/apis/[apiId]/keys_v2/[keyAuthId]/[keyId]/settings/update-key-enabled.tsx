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
import { Form, FormControl, FormField, FormItem, FormLabel } from "@/components/ui/form";
import { Switch } from "@/components/ui/switch";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  keyId: z.string(),
  workspaceId: z.string(),
  enabled: z.boolean(),
});
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    enabled: boolean;
  };
};

export const UpdateKeyEnabled: React.FC<Props> = ({ apiKey }) => {
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      keyId: apiKey.id,
      workspaceId: apiKey.workspaceId,
      enabled: apiKey.enabled,
    },
  });
  const updateEnabled = trpc.key.update.enabled.useMutation({
    onSuccess() {
      toast.success("Your key has been updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    await updateEnabled.mutateAsync(values);
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardHeader>
            <CardTitle>Enable Key</CardTitle>
            <CardDescription>
              Enable or disable this key. Disabled keys will not verify.
            </CardDescription>
          </CardHeader>
          <CardContent className="flex justify-between item-center">
            <div className="flex flex-col space-y-2">
              {/*  */}
              <FormField
                control={form.control}
                name="enabled"
                render={({ field }) => (
                  <FormItem className="w-full">
                    <div className="flex items-center gap-4">
                      <FormControl>
                        <Switch
                          id="enableSwitch"
                          checked={form.getValues("enabled")}
                          onCheckedChange={(e) => {
                            field.onChange(e);
                          }}
                        />
                      </FormControl>{" "}
                      <FormLabel htmlFor="enabled">
                        {form.getValues("enabled") ? "Enabled" : "Disabled"}
                      </FormLabel>
                    </div>
                  </FormItem>
                )}
              />
            </div>
          </CardContent>
          <CardFooter className="justify-end">
            <Button disabled={updateEnabled.isLoading || !form.formState.isValid} type="submit">
              {updateEnabled.isLoading ? <Loading /> : "Save"}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
