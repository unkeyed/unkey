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
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
const formSchema = z.object({
  keyAuthId: z.string(),
  defaultBytes: z
    .number()
    .min(8, "Key must be between 8 and 255 bytes long")
    .max(255, "Key must be between 8 and 255 bytes long")
    .optional(),
});

type Props = {
  keyAuth: {
    id: string;
    defaultBytes: number | undefined | null;
  };
};

export const DefaultBytes: React.FC<Props> = ({ keyAuth }) => {
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: async (data, context, options) => {
      return zodResolver(formSchema)(data, context, options);
    },
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      defaultBytes: keyAuth.defaultBytes ?? undefined,
      keyAuthId: keyAuth.id,
    },
  });

  const setDefaultBytes = trpc.api.setDefaultBytes.useMutation({
    onSuccess() {
      toast.success("Default Byte length for this API is updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (values.defaultBytes === keyAuth.defaultBytes || !values.defaultBytes) {
      return toast.error(
        "Please provide a different byte-size than already existing one as default",
      );
    }
    await setDefaultBytes.mutateAsync(values);
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <Card>
          <CardHeader>
            <CardTitle>Default Bytes</CardTitle>
            <CardDescription>
              Set default Bytes for the keys under this API. Default byte size must be between{" "}
              <span className="font-bold">8 to 255</span>
            </CardDescription>
          </CardHeader>
          <CardContent>
            <div className="flex flex-col space-y-2">
              <input type="hidden" name="keyAuthId" value={keyAuth.id} />
              <label htmlFor="defaultBytes" className="hidden sr-only">
                Default Bytes
              </label>
              <FormField
                control={form.control}
                name="defaultBytes"
                render={({ field }) => (
                  <FormItem>
                    <FormControl>
                      <Input
                        id="defaultBytes"
                        className="max-w-sm"
                        {...field}
                        autoComplete="off"
                        onChange={(e) => field.onChange(Number(e.target.value.replace(/\D/g, "")))}
                      />
                    </FormControl>
                    <FormMessage />
                  </FormItem>
                )}
              />
            </div>
          </CardContent>
          <CardFooter className="justify-end">
            <Button
              variant="primary"
              disabled={!form.formState.isValid || form.formState.isSubmitting}
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
