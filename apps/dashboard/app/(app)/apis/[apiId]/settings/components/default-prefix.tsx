"use client";
import { revalidate } from "@/app/actions";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { toast } from "@/components/ui/toaster";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, Input, SettingCard } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { keyPrefixSchema } from "../../_components/create-key/create-key.schema";

const formSchema = z.object({
  keyAuthId: z.string(),
  defaultPrefix: keyPrefixSchema.pipe(z.string()),
});

type Props = {
  keyAuth: {
    id: string;
    defaultPrefix: string | undefined | null;
  };
  apiId: string;
};

export const DefaultPrefix: React.FC<Props> = ({ keyAuth, apiId }) => {
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
      toast.success("Default Prefix Updated", {
        description: "Default prefix for this API has been successfully updated.",
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
            "We encountered an issue while updating the default prefix. Please try again later or contact support at support@unkey.dev",
        });
      } else {
        toast.error("Failed to Update Default Prefix", {
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
    if (values.defaultPrefix === keyAuth.defaultPrefix) {
      return toast.error("Please provide a different prefix than already existing one as default");
    }
    await setDefaultPrefix.mutateAsync(values);
    revalidate(`/apis/${apiId}/settings`);
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <SettingCard
          title={"Default Prefix"}
          description={
            <div className="max-w-[380px]">
              Sets the default prefix for keys under this API. A trailing underscore is added
              automatically.
            </div>
          }
          border="bottom"
          contentWidth="w-full lg:w-[420px] h-full"
        >
          <div className="flex flex-row justify-end items-center w-full gap-x-2">
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
                      className="w-[16.5rem]"
                      {...field}
                      autoComplete="off"
                      onBlur={(e) => {
                        if (e.target.value === "") {
                          return;
                        }
                      }}
                    />
                  </FormControl>
                </FormItem>
              )}
            />
            <Button
              variant="primary"
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
          </div>
        </SettingCard>
      </form>
    </Form>
  );
};
