"use client";

import { toast } from "@/components/ui/toaster";
import { tags } from "@/lib/cache";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, FormInput, SettingCard } from "@unkey/ui";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { revalidateTag } from "../../../../../actions";

export const dynamic = "force-dynamic";

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
  const utils = trpc.useUtils();

  const {
    control,
    handleSubmit,
    formState: { isValid, isSubmitting, isDirty },
  } = useForm<z.infer<typeof formSchema>>({
    resolver: zodResolver(formSchema),
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
      // Invalidate only the API overview query to update the sidebar
      utils.api.overview.query.invalidate();
      // No need for a full page reload
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
    <form onSubmit={handleSubmit(onSubmit)}>
      <SettingCard
        title="Name"
        description="Change the name of your API. This is only visible to you and your team."
        border="top"
        className="border-b-1"
        contentWidth="w-full lg:w-[420px]"
      >
        <div className="flex flex-row items-center w-full gap-x-2 justify-end">
          <input type="hidden" name="apiId" value={api.id} />
          <input type="hidden" name="workspaceId" value={api.workspaceId} />

          <Controller
            control={control}
            name="apiName"
            render={({ field }) => (
              <FormInput
                {...field}
                placeholder="my-api"
                className="w-[16.5rem]"
                onChange={(e) => {
                  if (e.target.value === "") {
                    return;
                  }
                  field.onChange(e);
                }}
              />
            )}
          />

          <Button
            size="lg"
            variant="primary"
            disabled={!isValid || isSubmitting || updateName.isLoading || !isDirty}
            type="submit"
            loading={isSubmitting || updateName.isLoading}
          >
            Save
          </Button>
        </div>
      </SettingCard>
    </form>
  );
};
