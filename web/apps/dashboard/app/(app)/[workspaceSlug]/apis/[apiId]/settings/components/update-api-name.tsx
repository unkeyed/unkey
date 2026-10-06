"use client";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import {
  FormInput,
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowDescription,
  SettingsRowHeader,
  SettingsRowTitle,
  formSaveState,
} from "@unkey/ui";
import type { Resolver } from "react-hook-form";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import {
  createApiFormConfig,
  createMutationHandlers,
  validateFormChange,
} from "./key-settings-form-helper";

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
    formState: { isValid, isSubmitting, isDirty, errors },
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
    reset(values);
  }

  return (
    <SettingsForm
      onSubmit={handleSubmit(onSubmit)}
      dirty={isDirty}
      saveState={formSaveState({ isSubmitting, isValid, isDirty })}
    >
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>Name</SettingsRowTitle>
          <SettingsRowDescription>
            Change the name of your keyspace. This is only visible to you and your team.
          </SettingsRowDescription>
        </SettingsRowHeader>
        <SettingsRowContent>
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
                className="max-w-(--setting-w)"
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
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
};
