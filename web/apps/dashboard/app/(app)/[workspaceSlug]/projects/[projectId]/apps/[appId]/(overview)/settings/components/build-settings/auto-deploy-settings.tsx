"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { IconHalfDottedCirclePlayOutline18 } from "nucleo-ui-outline-18";
import { useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { Switch } from "@/components/ui/switch";
import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
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
    // One transaction for both environments. The collection refetches every
    // loaded environment after a transaction settles.
    const targets: { id: string; autoDeploy: boolean }[] = [];
    if (values.production !== defaultProd) {
      targets.push({ id: production.environmentId, autoDeploy: values.production });
    }
    if (values.preview !== defaultPreview) {
      targets.push({ id: preview.environmentId, autoDeploy: values.preview });
    }
    if (targets.length === 0) {
      return;
    }

    collection.environmentSettings.update(
      targets.map((t) => t.id),
      (drafts) =>
        drafts.forEach((draft, i) => {
          draft.autoDeploy = targets[i].autoDeploy;
        }),
    );
  };

  const hasChanges = currentProd !== defaultProd || currentPreview !== defaultPreview;

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<IconHalfDottedCirclePlayOutline18 className="text-gray-12" />}
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
      footerLeft={
        <SettingDescription>
          When disabled, you can still deploy manually from the dashboard.
        </SettingDescription>
      }
    >
      <div className="flex flex-col gap-1" data-form-wide>
        <EnvRow
          label="Production"
          description="pushes to the default branch"
          checked={currentProd}
          onChange={(v) => setValue("production", v, { shouldValidate: true })}
        />
        <EnvRow
          label="Preview"
          description="pushes to non-default branches"
          checked={currentPreview}
          onChange={(v) => setValue("preview", v, { shouldValidate: true })}
        />
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
  <div className="flex items-center gap-3 py-1.5 cursor-pointer">
    <Switch checked={checked} onCheckedChange={onChange} size="sm" />
    <span className="text-sm text-gray-12">
      <span className="font-medium">{label}</span>
      <span className="text-gray-9"> — {description}</span>
    </span>
  </div>
);
