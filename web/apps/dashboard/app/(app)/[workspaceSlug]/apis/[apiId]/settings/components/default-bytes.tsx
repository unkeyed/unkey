"use client";
import { revalidate } from "@/app/actions";
import { useProjectScope } from "@/hooks/use-project-scope";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { FormInput, SettingsForm, SettingsRow, formSaveState } from "@unkey/ui";
import { Controller, useForm } from "react-hook-form";
import { z } from "zod";
import { keyBytesSchema } from "../../_components/create-key/create-key.schema";
import {
  createApiFormConfig,
  createMutationHandlers,
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
  const workspace = useWorkspaceNavigation();
  const scope = useProjectScope();

  const {
    control,
    handleSubmit,
    reset,
    formState: { isValid, isSubmitting, isDirty, errors },
  } = useForm<z.infer<typeof formSchema>>({
    ...createApiFormConfig(formSchema),
    // biome-ignore lint/suspicious/noExplicitAny: Zod v4 type inference with z.coerce creates resolver type mismatch
    resolver: zodResolver(formSchema) as any,
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
    reset(values);

    revalidate(routes.apis.settings({ workspaceSlug: workspace.slug, ...scope, apiId }));
  }

  return (
    <SettingsForm
      onSubmit={handleSubmit(onSubmit)}
      dirty={isDirty}
      saveState={formSaveState({ isSubmitting, isValid, isDirty })}
    >
      <SettingsRow
        title="Default Bytes"
        description="Sets the default byte size for keys under this keyspace. Must be between 16 and 255."
      >
        <input type="hidden" name="keyAuthId" value={keyAuth.id} />

        <Controller
          control={control}
          name="defaultBytes"
          render={({ field }) => (
            <FormInput
              {...field}
              aria-label="Default Bytes"
              autoComplete="off"
              type="text"
              error={errors.defaultBytes?.message}
              onChange={(e) => field.onChange(Number(e.target.value.replace(/\D/g, "")))}
            />
          )}
        />
      </SettingsRow>
    </SettingsForm>
  );
};
