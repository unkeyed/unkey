"use client";

import { Loading } from "@/components/dashboard/loading";
import { SettingCard } from "@/components/settings-card";
import { Form, FormControl, FormField, FormItem } from "@/components/ui/form";
import { Input } from "@/components/ui/input";
import { toast } from "@/components/ui/toaster";
import { tags } from "@/lib/cache";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button } from "@unkey/ui";
import { useRouter } from "next/navigation";
import { useForm } from "react-hook-form";
import { z } from "zod";
import { revalidateTag } from "../../../../../actions";

const formSchema = z.object({
  apiName: z.string().trim().min(3, "Name is required and should be at least 3 characters"),
  apiId: z.string(),
  workspaceId: z.string(),
});

type Props = {
  api: {
    id: string;
    workspaceId: string;
    name: string;
  };
};

export const UpdateApiName: React.FC<Props> = ({ api }) => {
  const router = useRouter();
  const form = useForm<z.infer<typeof formSchema>>({
    resolver: async (data, context, options) => {
      return zodResolver(formSchema)(data, context, options);
    },
    mode: "all",
    shouldFocusError: true,
    delayError: 100,
    defaultValues: {
      apiName: api.name,
      apiId: api.id,
      workspaceId: api.workspaceId,
    },
  });

  const updateName = trpc.api.updateName.useMutation({
    onSuccess() {
      toast.success("Your API name has been renamed!");
      revalidateTag(tags.api(api.id));
      router.refresh();
    },
    onError(err) {
      console.error(err);
      toast.error(err.message);
    },
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (values.apiName === api.name || !values.apiName) {
      return toast.error("Please provide a valid name before saving.");
    }
    await updateName.mutateAsync({
      name: values.apiName,
      apiId: values.apiId,
      workspaceId: values.workspaceId,
    });
  }

  return (
    <Form {...form}>
      <form onSubmit={form.handleSubmit(onSubmit)}>
        <SettingCard
          className="py-[19px] mt-5"
          title={
            <div className="flex items-center justify-start gap-2.5">
              <span className="text-sm font-medium text-accent-12">API Name</span>
            </div>
          }
          description={
            <div className="font-normal text-[13px] max-w-[380px]">
              Not customer-facing. Choose a name that is easy to recognize.
            </div>
          }
          border="top"
        >
          <div className="flex items-center w-full gap-2 sm:justify-start lg:justify-center">
            <input type="hidden" name="workspaceId" value={api.workspaceId} />
            <input type="hidden" name="apiId" value={api.id} />
            <label htmlFor="apiName" className="hidden sr-only">
              Name
            </label>
            <FormField
              control={form.control}
              name="apiName"
              render={({ field }) => (
                <FormItem>
                  <FormControl>
                    <Input
                      type="text"
                      id="apiName"
                      className="w-[257px] h-9"
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
              size="lg"
              className="rounded-lg px-2.5 "
              disabled={
                !form.formState.isValid ||
                form.formState.isSubmitting ||
                api.name === form.watch("apiName")
              }
              loading={form.formState.isSubmitting}
              type="submit"
            >
              {form.formState.isSubmitting ? <Loading /> : "Save"}
            </Button>
          </div>
        </SettingCard>
      </form>
    </Form>
  );
};
