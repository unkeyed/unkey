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
import { Switch } from "@/components/ui/switch";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { format } from "date-fns";

import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const currentTime = new Date();
const oneMinute = currentTime.setMinutes(currentTime.getMinutes() + 0.5);
const formSchema = z.object({
  keyId: z.string(),
  enableExpiration: z.boolean(),
  expiration: z.coerce.date().min(new Date(oneMinute)).optional(),
});
type Props = {
  apiKey: {
    id: string;
    workspaceId: string;
    expires: Date | null;
  };
};

export const UpdateKeyExpiration: React.FC<Props> = ({ apiKey }) => {
  const router = useRouter();

  /*  This ensures the date shown is in local time and not ISO  */
  function convertDate(date: Date | null): string {
    if (!date) {
      return "";
    }
    return format(date, "yyyy-MM-dd'T'HH:mm:ss");
  }

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      keyId: apiKey.id ? apiKey.id : undefined,
      enableExpiration: apiKey.expires !== null,
    },
  });

  const changeExpiration = trpc.key.update.expiration.useMutation({
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
    await changeExpiration.mutateAsync({
      keyId: values.keyId,
      expiration: { enabled: values.enableExpiration, data: values.expiration },
    });
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardHeader>
            <CardTitle>Expiration</CardTitle>
            <CardDescription>Automatically revoke this key after a certain date.</CardDescription>
          </CardHeader>
          <CardContent className="flex justify-between item-center">
            <div
              className={cn("flex flex-col gap-2 w-full", {
                "opacity-50": !form.getValues("enableExpiration"),
              })}
            >
              <FormField
                control={form.control}
                name="expiration"
                render={({ field }) => (
                  <FormItem className="w-fit">
                    <FormLabel>Expiry Date</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        disabled={!form.watch("enableExpiration")}
                        type="datetime-local"
                        value={field.value?.toLocaleString()}
                        defaultValue={convertDate(apiKey.expires)}
                      />
                    </FormControl>
                    <FormDescription>
                      This api key will automatically be revoked after the given date.
                    </FormDescription>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </CardContent>
          <CardFooter className="justify-between">
            <FormField
              control={form.control}
              name="enableExpiration"
              render={({ field }) => (
                <FormItem className="w-full">
                  <div className="flex items-center gap-4">
                    <FormControl>
                      <Switch
                        id="enableExpiration"
                        checked={form.getValues("enableExpiration")}
                        onCheckedChange={(e) => {
                          field.onChange(e);
                        }}
                      />
                    </FormControl>{" "}
                    <FormLabel htmlFor="enableExpiration">
                      {form.getValues("enableExpiration") ? "Enabled" : "Disabled"}
                    </FormLabel>
                  </div>
                </FormItem>
              )}
            />
            <Button disabled={changeExpiration.isLoading || !form.formState.isValid} type="submit">
              {changeExpiration.isLoading ? <Loading /> : "Save"}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
