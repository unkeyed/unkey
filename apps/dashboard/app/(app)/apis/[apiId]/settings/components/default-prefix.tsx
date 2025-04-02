"use client";
import { SettingCard } from "@/components/settings-card";
import { Form, FormControl, FormField, FormItem, FormMessage } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { InputPasswordSettings } from "@unkey/icons";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";

const formSchema = z.object({
  keyAuthId: z.string(),
  defaultPrefix: z
    .string()
    .max(8, { message: "Prefixes cannot be longer than 8 characters" })
    .refine((prefix) => !prefix.includes(" "), {
      message: "Prefixes cannot contain spaces.",
    }),
});

type Props = {
  keyAuth: {
    id: string;
    defaultPrefix: string | undefined | null;
  };
};

export const DefaultPrefix: React.FC<Props> = ({ keyAuth }) => {
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: async (data, context, options) => {
      return zodResolver(formSchema)(data, context, options);
    },
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      defaultPrefix: keyAuth.defaultPrefix ?? undefined,
      keyAuthId: keyAuth.id,
    },
  });

  const setDefaultPrefix = trpc.api.setDefaultPrefix.useMutation({
    onSuccess() {
      toast.success("Default prefix for this API is updated!");
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });
  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (values.defaultPrefix === keyAuth.defaultPrefix) {
      return toast.error("Please provide a different prefix than already existing one as default");
    }
    await setDefaultPrefix.mutateAsync(values);
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <SettingCard
          className="py-5"
          title={
            <div className=" flex items-center justify-start gap-2.5">
              <InputPasswordSettings size="xl-medium" className="h-full text-brand-10" />
              <span className="text-sm font-medium text-accent-12">Default Prefix</span>
            </div>
          }
          description={
            <div>
              Sets the default prefix for keys under this API. A trailing <br /> underscore is added
              automatically.
            </div>
          }
          border="bottom"
        >
          <div className="items-center justify-center w-full gap-2 lex">
            <input type="hidden" name="keyAuthId" value={keyAuth.id} />
            <label htmlFor="defaultPrefix" className="hidden sr-only">
              Default Prefix
            </label>
            <FormField
              control={form.control}
              name="defaultPrefix"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Input
                      id="defaultPrefix"
                      className="h-9 w-[257px]"
                      {...field}
                      onBlur={(e) => {
                        if (e.target.value === "") {
                          return;
                        }
                      }}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
          </div>
          <Button
            className="rounded-lg px-2.5"
            size="lg"
            disabled={
              !form.formState.isValid ||
              form.formState.isSubmitting ||
              keyAuth.defaultPrefix === form.getValues("defaultPrefix")
            }
            type="submit"
            loading={form.formState.isSubmitting}
          >
            Save
          </Button>
        </SettingCard>
      </form>
    </Form>
  );
};
