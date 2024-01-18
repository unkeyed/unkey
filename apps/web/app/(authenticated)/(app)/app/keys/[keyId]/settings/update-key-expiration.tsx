"use client";
import { Loading } from "@/components/dashboard/loading";
import { Button } from "@/components/ui/button";
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
import { Label } from "@/components/ui/label";
import { Switch } from "@/components/ui/switch";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { cn } from "@/lib/utils";
import { zodResolver } from "@hookform/resolvers/zod";
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
  // const [enabled, setEnabled] = useState(apiKey.expires !== null);
  const router = useRouter();

  function getDatePlusTwoMinutes(): string {
    if (apiKey.expires) {
      return apiKey.expires.toISOString().slice(0, -8);
    }
    const now = new Date();
    const futureDate = new Date(now.getTime() + 2 * 60000);
    return futureDate.toISOString().slice(0, -8);
  }
  // const placeholder = useMemo(() => {
  //   const t = new Date();
  //   t.setUTCDate(t.getUTCDate() + 7);
  //   t.setUTCMinutes(0, 0, 0);
  //   return t.toISOString();
  // }, []);

  const form = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      keyId: apiKey.id ? apiKey.id : undefined,
      enableExpiration: apiKey.expires !== null ? true : false,
      expiration: apiKey.expires ? apiKey.expires : undefined,
    },
  });

  const changeExpiration = trpc.keySettings.updateExpiration.useMutation({
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
    changeExpiration.mutateAsync(values);
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
              <Label htmlFor="expiration">Expiration</Label>
              <FormField
                control={form.control}
                name="expiration"
                render={({ field }) => (
                  <FormItem>
                    <FormLabel>Expiry Date</FormLabel>
                    <FormControl>
                      <Input
                        {...field}
                        disabled={!form.watch("enableExpiration")}
                        min={getDatePlusTwoMinutes()}
                        type="datetime-local"
                        // defaultValue={getDatePlusTwoMinutes()}
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
            <Button
              disabled={form.formState.isSubmitting || !form.formState.isValid}
              className="mt-4 "
              type="submit"
            >
              {form.formState.isSubmitting ? <Loading /> : "Save"}
            </Button>
          </CardFooter>
        </Card>
      </form>
    </Form>
  );
};
