"use client";

import { Switch } from "@/components/ui/switch";
import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { zodResolver } from "@hookform/resolvers/zod";
import { HalfDottedCirclePlay } from "@unkey/icons";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useMultiEnvironmentSettings } from "../../hooks/use-multi-environment-settings";
import { SettingDescription } from "../shared/form-blocks";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";

const dualSchema = z.object({ production: z.boolean(), preview: z.boolean() });
type DualFormValues = z.infer<typeof dualSchema>;

export const AutoDeploy = () => {
  const multiSettings = useMultiEnvironmentSettings();
  if (!multiSettings) {
    return null;
  }
  return <AutoDeployInner production={multiSettings.production} preview={multiSettings.preview} />;
};

const AutoDeployInner = ({
  production,
  preview,
}: {
  production: EnvironmentSettings;
  preview: EnvironmentSettings;
}) => {
  const defaultProd = production.autoDeploy;
  const defaultPreview = preview.autoDeploy;

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<DualFormValues>({
    resolver: zodResolver(dualSchema),
    mode: "onChange",
    defaultValues: { production: defaultProd, preview: defaultPreview },
  });

  useEffect(() => {
    reset({ production: defaultProd, preview: defaultPreview });
  }, [defaultProd, defaultPreview, reset]);

  const currentProd = useWatch({ control, name: "production" });
  const currentPreview = useWatch({ control, name: "preview" });

  const onSubmit = async (values: DualFormValues) => {
    if (values.production !== defaultProd) {
      collection.environmentSettings.update(production.environmentId, (draft) => {
        draft.autoDeploy = values.production;
      });
    }
    if (values.preview !== defaultPreview) {
      collection.environmentSettings.update(preview.environmentId, (draft) => {
        draft.autoDeploy = values.preview;
      });
    }
  };

  const hasChanges = currentProd !== defaultProd || currentPreview !== defaultPreview;

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<HalfDottedCirclePlay className="text-gray-12" iconSize="xl-medium" />}
      title="Auto deploy"
      description="Automatically trigger deployments when code is pushed to GitHub."
      displayValue={
        <div className="flex items-center gap-3">
          <span className="space-x-1">
            <span className="text-gray-11 text-xs font-normal">Production</span>
            <span className="font-medium text-gray-12">{defaultProd ? "On" : "Off"}</span>
          </span>
          <span className="text-gray-8">|</span>
          <span className="space-x-1">
            <span className="text-gray-11 text-xs font-normal">Preview</span>
            <span className="font-medium text-gray-12">{defaultPreview ? "On" : "Off"}</span>
          </span>
        </div>
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <div className="flex flex-col gap-3 px-4 py-4">
        <div className="flex flex-col gap-2">
          <EnvRow
            label="Production"
            description="Pushes to the default branch"
            checked={currentProd}
            onChange={(v) => setValue("production", v, { shouldValidate: true })}
          />
          <EnvRow
            label="Preview"
            description="Pushes to non-default branches"
            checked={currentPreview}
            onChange={(v) => setValue("preview", v, { shouldValidate: true })}
          />
        </div>
        <SettingDescription>
          When disabled, you can still deploy manually from the dashboard.
        </SettingDescription>
      </div>
    </FormSettingCard>
  );
};

const EnvRow = ({
  label,
  description,
  checked,
  onChange,
}: {
  label: string;
  description: string;
  checked: boolean;
  onChange: (value: boolean) => void;
}) => (
  <div className="flex items-center gap-4 px-1 py-0.5">
    <div className="flex flex-col w-44">
      <span className="text-[13px] text-gray-12">{label}</span>
      <span className="text-xs text-gray-9">{description}</span>
    </div>
    <Switch
      checked={checked}
      onCheckedChange={onChange}
      className="h-5 w-10 data-[state=checked]:bg-info-9 data-[state=checked]:ring-2 data-[state=checked]:ring-infoA-5 data-[state=unchecked]:bg-grayA-6 data-[state=unchecked]:ring-2 data-[state=unchecked]:ring-grayA-5"
      thumbClassName="h-4 w-4 data-[state=checked]:translate-x-5"
    />
  </div>
);
