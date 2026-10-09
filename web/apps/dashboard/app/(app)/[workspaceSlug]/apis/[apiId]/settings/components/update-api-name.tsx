"use client";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { FormInput, SettingsForm, SettingsRow, formSaveState } from "@unkey/ui";
import type { Resolver } from "react-hook-form";
import { Controller, useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { createApiFormConfig, createMutationHandlers } from "./key-settings-form-helper";

export const dynamic = "force-dynamic";

const formSchema = z.object({
  apiName: z
    .string()
    .trim()
    .min(3, "Name is required and should be at least 3 characters")
    .max(256, "Name cannot exceed 256 characters"),
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
    reset,
    formState: { isValid, isSubmitting, isDirty: isFormDirty, errors },
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
    onSuccess: onUpdateSuccess("Keyspace name updated successfully"),
    onError,
  });

  const apiName = useWatch({ control, name: "apiName" });
  const isDirty = isFormDirty && apiName.trim() !== api.name.trim();

  async function onSubmit(values: z.infer<typeof formSchema>) {
    await updateName.mutateAsync({
      name: values.apiName,
      apiId: values.apiId,
      workspaceId: values.workspaceId,
    });
    reset(values);
  }

  return (
    <SettingsForm
      onSubmit={handleSubmit(onSubmit)}
      dirty={isDirty}
      saveState={formSaveState({ isSubmitting, isValid, isDirty })}
    >
      <SettingsRow
        title="Name"
        description="Change the name of your keyspace. This is only visible to you and your team."
      >
        <input type="hidden" name="apiId" value={api.id} />
        <input type="hidden" name="workspaceId" value={api.workspaceId} />

        <Controller
          control={control}
          name="apiName"
          render={({ field }) => (
            <FormInput
              {...field}
              aria-label="Name"
              placeholder="my-keyspace"
              error={errors.apiName?.message}
              onChange={(e) => {
                if (e.target.value === "") {
                  return;
                }
                field.onChange(e);
              }}
            />
          )}
        />
      </SettingsRow>
    </SettingsForm>
  );
};
