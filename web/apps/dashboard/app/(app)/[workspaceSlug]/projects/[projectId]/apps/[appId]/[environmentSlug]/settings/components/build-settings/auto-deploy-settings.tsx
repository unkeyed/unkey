"use client";

import { Switch } from "@/components/ui/switch";
import { ENVIRONMENT_KIND } from "@/lib/collections/deploy/environments";
import { zodResolver } from "@hookform/resolvers/zod";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useProjectData } from "../../../../data-provider";
import { useEnvironmentSettings } from "../../environment-provider";
import { useUpdateEnvironment } from "../../hooks/use-update-environment";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const schema = z.object({ autoDeploy: z.boolean() });
type FormValues = z.infer<typeof schema>;

export const AutoDeploy = () => {
  const { settings } = useEnvironmentSettings();
  const { environments } = useProjectData();
  const updateEnvironment = useUpdateEnvironment();
  const kind = environments.find((e) => e.id === settings.environmentId)?.kind;
  const trigger =
    kind === ENVIRONMENT_KIND.production
      ? "pushes to the default branch"
      : "pushes to non-default branches";
  const defaultValue = settings.autoDeploy;

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<FormValues>({
    resolver: zodResolver(schema),
    mode: "onChange",
    defaultValues: { autoDeploy: defaultValue },
  });

  useEffect(() => {
    reset({ autoDeploy: defaultValue });
  }, [defaultValue, reset]);

  const current = useWatch({ control, name: "autoDeploy" });

  const onSubmit = async (values: FormValues) => {
    updateEnvironment((draft) => {
      draft.autoDeploy = values.autoDeploy;
    });
  };

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [current === defaultValue, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      title="Auto deploy"
      description="Automatically trigger deployments when code is pushed to GitHub. When off, you can still deploy by hand from the dashboard."
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <div className="flex items-center gap-3 py-1.5">
        <Switch
          checked={current}
          onCheckedChange={(v) => setValue("autoDeploy", v, { shouldValidate: true })}
          size="sm"
        />
        <span className="text-sm text-gray-12">
          <span className="font-medium">Deploy</span>
          <span className="text-gray-9"> — {trigger}</span>
        </span>
      </div>
    </FormSettingCard>
  );
};
