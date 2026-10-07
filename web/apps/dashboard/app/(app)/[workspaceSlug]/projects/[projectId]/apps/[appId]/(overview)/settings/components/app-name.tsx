"use client";

import { collection } from "@/lib/collections";
import { createAppRequestSchema } from "@/lib/collections/deploy/apps";
import { getErrorMessage, getUnkeyClient } from "@/lib/unkey-client";
import { zodResolver } from "@hookform/resolvers/zod";
import { FormInput, formSaveState, toast } from "@unkey/ui";
import { useForm, useWatch } from "react-hook-form";
import type { z } from "zod";
import { SettingField } from "./shared/form-blocks";
import { FormSettingCard } from "./shared/form-setting-card";

const appNameSchema = createAppRequestSchema.pick({ name: true });

type AppNameFormValues = z.infer<typeof appNameSchema>;

export function AppName({
  projectId,
  appId,
  name,
}: { projectId: string; appId: string; name: string }) {
  const {
    register,
    handleSubmit,
    reset,
    control,
    formState: { errors, isSubmitting, isValid },
  } = useForm<AppNameFormValues>({
    resolver: zodResolver(appNameSchema),
    mode: "onChange",
    values: { name },
  });

  const currentName = useWatch({ control, name: "name" });
  const dirty = currentName.trim() !== name;
  const saveState = formSaveState({ isSubmitting, isValid, isDirty: dirty });

  const onSubmit = async (values: AppNameFormValues) => {
    try {
      await getUnkeyClient().apps.updateApp({ project: projectId, app: appId, name: values.name });
      reset(values);
      await collection.apps.utils.refetch();
      toast.success("App renamed");
    } catch (error) {
      toast.error(getErrorMessage(error, "Failed to rename app"));
    }
  };

  return (
    <FormSettingCard
      title="Name"
      onSubmit={handleSubmit(onSubmit)}
      dirty={dirty}
      saveState={saveState}
    >
      <SettingField>
        <FormInput
          data-1p-ignore
          aria-label="App name"
          error={errors.name?.message}
          {...register("name")}
        />
      </SettingField>
    </FormSettingCard>
  );
}
