"use client";
import { revalidate } from "@/app/actions";
import { trpc } from "@/lib/trpc/client";
import { Button, Input, SettingCard } from "@unkey/ui";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { keyPrefixSchema } from "../../_components/create-key/create-key.schema";
import {
  createApiFormConfig,
  createMutationHandlers,
  getStandardButtonProps,
  validateFormChange,
} from "./key-settings-form-helper";

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
  workspaceSlug: string;
};

export const DefaultPrefix: React.FC<Props> = ({ keyAuth, apiId, workspaceSlug }) => {
  const { onUpdateSuccess, onError } = createMutationHandlers();

  const {
    control,
    handleSubmit,
    formState: { isValid, isSubmitting, isDirty },
  } = useForm<z.infer<typeof formSchema>>({
    ...createApiFormConfig(formSchema),
    defaultValues: {
      defaultPrefix: keyAuth.defaultPrefix ?? undefined,
      keyAuthId: keyAuth.id,
    },
  });

  const setDefaultPrefix = trpc.api.setDefaultPrefix.useMutation({
    onSuccess: onUpdateSuccess("Default Prefix Updated"),
    onError,
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (
      !validateFormChange(
        keyAuth.defaultPrefix,
        values.defaultPrefix,
        "Please provide a different prefix than already existing one as default",
      )
    ) {
      return;
    }

    await setDefaultPrefix.mutateAsync(values);
    revalidate(`/${workspaceSlug}/apis/${apiId}/settings`);
  }

  return (
    <SettingCard
      title="Default Prefix"
      description={
        <div className="max-w-[380px]">
          Sets the default prefix for keys under this API. A trailing underscore is added
          automatically.
        </div>
      }
      border="bottom"
      contentWidth="w-full lg:w-[420px] h-full justify-end items-end"
    >
      <form
        onSubmit={handleSubmit(onSubmit)}
        className="flex flex-row justify-end items-center gap-x-2 h-9"
      >
        <input type="hidden" name="keyAuthId" value={keyAuth.id} />

        <Controller
          control={control}
          name="defaultPrefix"
          render={({ field }) => (
            <Input
              {...field}
              className="min-w-[16rem] items-end h-9"
              autoComplete="off"
              onBlur={(e) => {
                if (e.target.value === "") {
                  return;
                }
                field.onBlur();
              }}
            />
          )}
        />

        <Button {...getStandardButtonProps(isValid, isSubmitting, isDirty)}>Save</Button>
      </form>
    </SettingCard>
  );
};
