"use client";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { Button, Input, SettingCard } from "@unkey/ui";
import type { Resolver } from "react-hook-form";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import {
  createApiFormConfig,
  createMutationHandlers,
  getStandardButtonProps,
  validateFormChange,
} from "./key-settings-form-helper";

export const dynamic = "force-dynamic";

const formSchema = z.object({
  apiName: z.string().trim().min(3, "Name is required and should be at least 3 characters"),
  apiId: z.string(),
  workspaceId: z.string(),
});

type FormValues = z.infer<typeof formSchema>;

type Props = {
  api: {
    id: string;
    workspaceId: string;
    name: string;
  };
};

export const UpdateApiName: React.FC<Props> = ({ api }) => {
  const { onUpdateSuccess, onError } = createMutationHandlers();

  const {
    control,
    handleSubmit,
    formState: { isValid, isSubmitting, isDirty },
  } = useForm<FormValues>({
    ...createApiFormConfig(formSchema),
    resolver: zodResolver(formSchema) as Resolver<FormValues>,
    defaultValues: {
      apiName: api.name,
      apiId: api.id,
      workspaceId: api.workspaceId,
    },
  });

  const updateName = trpc.api.updateName.useMutation({
    onSuccess: onUpdateSuccess("API name updated successfully"),
    onError,
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (
      !validateFormChange(api.name, values.apiName, "Please provide a valid name before saving.")
    ) {
      return;
    }

    await updateName.mutateAsync({
      name: values.apiName,
      apiId: values.apiId,
      workspaceId: values.workspaceId,
    });
  }

  return (
    <SettingCard
      title="Name"
      description="Change the name of your API. This is only visible to you and your team."
      border="top"
      contentWidth="w-full lg:w-[420px] h-full justify-end items-end"
    >
      <form
        onSubmit={handleSubmit(onSubmit)}
        className="flex flex-row justify-end items-center gap-x-2 h-9"
      >
        <input type="hidden" name="apiId" value={api.id} />
        <input type="hidden" name="workspaceId" value={api.workspaceId} />

        <Controller
          control={control}
          name="apiName"
          render={({ field }) => (
            <Input
              {...field}
              placeholder="my-api"
              className="min-w-64 items-end h-9"
              onChange={(e) => {
                if (e.target.value === "") {
                  return;
                }
                field.onChange(e);
              }}
            />
          )}
        />

        <Button {...getStandardButtonProps(isValid, isSubmitting, isDirty)}>Save</Button>
      </form>
    </SettingCard>
  );
};
