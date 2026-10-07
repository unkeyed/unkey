"use client";
import { revalidate } from "@/app/actions";
import { useProjectScope } from "@/hooks/use-project-scope";
import { useWorkspaceNavigation } from "@/hooks/use-workspace-navigation";
import { routes } from "@/lib/navigation/routes";
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
import { keyPrefixSchema } from "../../_components/create-key/create-key.schema";
import {
  createApiFormConfig,
  createMutationHandlers,
  validateFormChange,
} from "./key-settings-form-helper";

const formSchema = z.object({
  keyAuthId: z.string(),
  defaultPrefix: keyPrefixSchema.pipe(z.string()),
});

type FormValues = z.infer<typeof formSchema>;

type Props = {
  keyAuth: {
    id: string;
    defaultPrefix: string | undefined | null;
  };
  apiId: string;
};

export const DefaultPrefix: React.FC<Props> = ({ keyAuth, apiId }) => {
  const { onUpdateSuccess, onError } = createMutationHandlers();
  const workspace = useWorkspaceNavigation();
  const scope = useProjectScope();

  const {
    control,
    handleSubmit,
    reset,
    formState: { isValid, isSubmitting, isDirty, errors },
  } = useForm<FormValues>({
    ...createApiFormConfig(formSchema),
    resolver: zodResolver(formSchema) as Resolver<FormValues>,
    defaultValues: {
      defaultPrefix: keyAuth.defaultPrefix ?? "",
      keyAuthId: keyAuth.id,
    },
  });

  const setDefaultPrefix = trpc.api.setDefaultPrefix.useMutation({
    onSuccess: onUpdateSuccess("Default Prefix Updated"),
    onError,
  });

  async function onSubmit(values: z.infer<typeof formSchema>) {
    if (!workspace) {
      return;
    }
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
    reset(values);
    revalidate(routes.apis.settings({ workspaceSlug: workspace.slug, ...scope, apiId }));
  }

  return (
    <SettingsForm
      onSubmit={handleSubmit(onSubmit)}
      dirty={isDirty}
      saveState={formSaveState({ isSubmitting, isValid, isDirty })}
    >
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>Default Prefix</SettingsRowTitle>
          <SettingsRowDescription>
            Sets the default prefix for keys under this keyspace. A trailing underscore is added
            automatically.
          </SettingsRowDescription>
        </SettingsRowHeader>
        <SettingsRowContent>
          <input type="hidden" name="keyAuthId" value={keyAuth.id} />

          <Controller
            control={control}
            name="defaultPrefix"
            render={({ field }) => (
              <FormInput
                {...field}
                aria-label="Default Prefix"
                className="max-w-(--setting-w)"
                autoComplete="off"
                error={errors.defaultPrefix?.message}
                onBlur={(e) => {
                  if (e.target.value === "") {
                    return;
                  }
                  field.onBlur();
                }}
              />
            )}
          />
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
};
