"use client";
import { SettingCard } from "@/components/settings-card";
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
        <SettingCard
          title={
            <div className="flex items-center justify-start gap-2.5">
              <span className="text-sm font-medium text-accent-12">Default Bytes</span>
            </div>
          }
          description={
            <div className="font-normal text-[13px] max-w-[380px]">
              Sets the default byte size for keys under this API. Must be between 8 and 255.
            </div>
          }
          border="top"
          contentWidth="w-full lg:w-[320px]"
        >
          <div className="flex flex-row justify-items-stretch items-center w-full gap-x-2">
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
                      className="w-[20rem] lg:w-[16rem] h-9"
                      {...field}
                      autoComplete="off"
                      onChange={(e) => field.onChange(Number(e.target.value.replace(/\D/g, "")))}
                    />
                  </FormControl>
                  <FormMessage />
                </FormItem>
              )}
            />
            <Button
              size="lg"
              variant="primary"
              disabled={
                !form.formState.isValid ||
                form.formState.isSubmitting ||
                keyAuth.defaultBytes === form.watch("defaultBytes")
              }
              type="submit"
              loading={form.formState.isSubmitting}
            >
              Save
            </Button>
          </div>
        </SettingCard>
      </form>
    </Form>
  );
};
