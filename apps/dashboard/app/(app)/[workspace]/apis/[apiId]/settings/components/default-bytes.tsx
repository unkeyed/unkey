"use client";
import { revalidate } from "@/app/actions";
import { trpc } from "@/lib/trpc/client";
import { useWorkspace } from "@/providers/workspace-provider";
import { Button, Input, SettingCard } from "@unkey/ui";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { keyBytesSchema } from "../../_components/create-key/create-key.schema";
import {
  createApiFormConfig,
  createMutationHandlers,
  getStandardButtonProps,
  validateFormChange,
} from "./key-settings-form-helper";

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
  const { onUpdateSuccess, onError } = createMutationHandlers();
  const { workspace } = useWorkspace();
  if (!workspace) {
    return null;
  }
  const {
    control,
    handleSubmit,
    formState: { isValid, isSubmitting, isDirty },
  } = useForm<z.infer<typeof formSchema>>({
    ...createApiFormConfig(formSchema),
    defaultValues: {
      defaultBytes: keyAuth.defaultBytes ?? undefined,
      keyAuthId: keyAuth.id,
    },
  });

  const setDefaultBytes = trpc.api.setDefaultBytes.useMutation({
    onSuccess: onUpdateSuccess("Default Byte Length Updated"),
    onError,
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (
      !validateFormChange(
        keyAuth.defaultBytes,
        values.defaultBytes,
        "Please provide a different byte-size than already existing one as default",
      )
    ) {
      return;
    }

    await setDefaultBytes.mutateAsync(values);
    revalidate(`/${workspace?.slug}/apis/${apiId}/settings`);
  }

  return (
    <SettingCard
      title="Default Bytes"
      description={
        <div className="max-w-[380px]">
          Sets the default byte size for keys under this API. Must be between 8 and 255.
        </div>
      }
      border="top"
      className="border-b"
      contentWidth="w-full lg:w-[420px] h-full justify-end items-end"
    >
      <form
        onSubmit={handleSubmit(onSubmit)}
        className="flex flex-row justify-end items-center gap-x-2 h-9"
      >
        <input type="hidden" name="keyAuthId" value={keyAuth.id} />

        <Controller
          control={control}
          name="defaultBytes"
          render={({ field }) => (
            <Input
              {...field}
              className="min-w-[16rem] items-end h-9"
              autoComplete="off"
              type="text"
              onChange={(e) => field.onChange(Number(e.target.value.replace(/\D/g, "")))}
            />
          )}
        />

        <Button {...getStandardButtonProps(isValid, isSubmitting, isDirty)}>Save</Button>
      </form>
    </SettingCard>
  );
};
