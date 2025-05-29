"use client";
import { revalidate } from "@/app/actions";
import { Form, FormControl, FormField, FormItem, FormMessage } from "@/components/ui/form";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, Input, SettingCard } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { keyBytesSchema } from "../../_components/create-key/create-key.schema";

const formSchema = z.object({
  keyAuthId: z.string(),
  defaultBytes: keyBytesSchema,
});

type Props = {
  keyAuth: {
    id: string;
    defaultBytes: number | undefined | null;
  };
  apiId: string;
};

export const DefaultBytes: React.FC<Props> = ({ keyAuth, apiId }) => {
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
      toast.success("Default Byte Length Updated", {
        description: "Default byte length for this API has been successfully updated.",
      });
      router.refresh();
    },
    onError(err) {
      console.error(err);

      if (err.data?.code === "NOT_FOUND") {
        toast.error("API Configuration Not Found", {
          description:
            "Unable to find the correct API configuration. Please refresh and try again.",
        });
      } else if (err.data?.code === "INTERNAL_SERVER_ERROR") {
        toast.error("Server Error", {
          description:
            "We encountered an issue while updating the default bytes. Please try again later or contact support at support@unkey.dev",
        });
      } else if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid Configuration", {
          description: `Please check your byte length settings. ${err.message || ""}`,
        });
      } else {
        toast.error("Failed to Update Default Bytes", {
          description: err.message || "An unexpected error occurred. Please try again later.",
          action: {
            label: "Contact Support",
            onClick: () => window.open("https://support.unkey.dev", "_blank"),
          },
        });
      }
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (values.defaultBytes === keyAuth.defaultBytes || !values.defaultBytes) {
      return toast.error(
        "Please provide a different byte-size than already existing one as default",
      );
    }
    await setDefaultBytes.mutateAsync(values);
    revalidate(`/apis/${apiId}/settings`);
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <SettingCard
          title={"Default Bytes"}
          description={
            <div className="max-w-[380px]">
              Sets the default byte size for keys under this API. Must be between 8 and 255.
            </div>
          }
          border="top"
          className="border-b-1"
          contentWidth="w-full lg:w-[420px]"
        >
          <div className="flex flex-row justify-end items-center w-full gap-x-2">
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
                      className="w-[16.5rem]"
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
