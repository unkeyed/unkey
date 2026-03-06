"use client";

import { collection } from "@/lib/collections";
import type { EnvironmentSettings } from "@/lib/collections/deploy/environment-settings";
import { mapRegionToFlag } from "@/lib/trpc/routers/deploy/network/utils";
import { zodResolver } from "@hookform/resolvers/zod";
import { Connections3 } from "@unkey/icons";
import { Slider } from "@unkey/ui";
import { useContext, useEffect } from "react";
import { useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { RegionFlag } from "../../../../components/region-flag";
import { EnvironmentContext, useEnvironmentSettings } from "../../environment-provider";
import { useMultiEnvironmentSettings } from "../../hooks/use-multi-environment-settings";
import { EnvironmentSliderSection } from "../shared/environment-slider-section";
import { FormSettingCard, resolveSaveState } from "../shared/form-setting-card";
import { SettingDescription } from "../shared/setting-description";

export const Instances = () => {
  const envContext = useContext(EnvironmentContext);

  if (envContext?.variant === "onboarding") {
    return <InstancesSingle />;
  }

  return <InstancesDual />;
};

const instancesSingleSchema = z.object({ instances: z.number().min(1).max(10) });
type InstancesSingleFormValues = z.infer<typeof instancesSingleSchema>;

const InstancesSingle = () => {
  const { settings, variant } = useEnvironmentSettings();
  const regions = settings.regions.map((r) => r.name);
  const defaultInstances = settings.regions[0]?.replicas ?? 1;

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<InstancesSingleFormValues>({
    resolver: zodResolver(instancesSingleSchema),
    mode: "onChange",
    defaultValues: { instances: defaultInstances },
  });

  useEffect(() => {
    reset({ instances: defaultInstances });
  }, [defaultInstances, reset]);

  const currentInstances = useWatch({ control, name: "instances" });

  const onSubmit = async (values: InstancesSingleFormValues) => {
    collection.environmentSettings.update(settings.environmentId, (draft) => {
      for (const region of draft.regions) {
        region.replicas = values.instances;
      }
    });
  };

  const hasChanges = currentInstances !== defaultInstances;
  const hasRegions = regions.length > 0;

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [
      !hasRegions,
      { status: "disabled", reason: "Select at least one region before setting instance count" },
    ],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<Connections3 className="text-gray-12" iconSize="xl-medium" />}
      title="Instances"
      description="Number of instances running in each region"
      displayValue={
        <span>
          <span className="font-medium text-gray-12">{defaultInstances}</span>{" "}
          <span className="text-gray-11">instance{defaultInstances !== 1 ? "s" : ""}</span>
        </span>
      }
      onSubmit={handleSubmit(onSubmit)}
      autoSave
      saveState={saveState}
    >
      <div className="flex items-center gap-3">
        <Slider
          min={1}
          max={10}
          step={1}
          value={[currentInstances]}
          onValueChange={([value]) => {
            if (value !== undefined) {
              setValue("instances", value, { shouldValidate: true });
            }
          }}
          onValueCommit={
            variant === "onboarding"
              ? ([value]) => {
                if (value !== undefined && value !== defaultInstances) {
                  collection.environmentSettings.update(settings.environmentId, (draft) => {
                    for (const region of draft.regions) {
                      region.replicas = value;
                    }
                  });
                }
              }
              : undefined
          }
          className="flex-1 max-w-[480px]"
          rangeStyle={{
            background:
              "linear-gradient(to right, hsla(var(--featureA-4)), hsla(var(--featureA-12)))",
            backgroundSize: `${currentInstances > 1 ? 100 / ((currentInstances - 1) / 9) : 100}% 100%`,
            backgroundRepeat: "no-repeat",
          }}
        />
        <div className="flex items-center gap-1.5">
          {regions.map((r) => (
            <RegionFlag
              key={r}
              flagCode={mapRegionToFlag(r)}
              size="xs"
              shape="circle"
              className="[&_img]:size-3"
            />
          ))}
        </div>
        <span className="text-[13px]">
          <span className="font-medium text-gray-12">{currentInstances}</span>{" "}
          <span className="text-gray-11 font-normal">
            instance{currentInstances !== 1 ? "s" : ""}
          </span>
        </span>
      </div>

      <SettingDescription>
        More instances improve availability and handle higher traffic. Changes apply on next deploy.
      </SettingDescription>
    </FormSettingCard>
  );
};

const instancesDualSchema = z.object({
  productionInstances: z.number().min(1).max(10),
  previewInstances: z.number().min(1).max(10),
});

type InstancesDualFormValues = z.infer<typeof instancesDualSchema>;

const InstancesDual = () => {
  const multiSettings = useMultiEnvironmentSettings();

  if (!multiSettings) {
    return null;
  }

  return (
    <InstancesDualInner production={multiSettings.production} preview={multiSettings.preview} />
  );
};

type InstancesDualInnerProps = {
  production: EnvironmentSettings;
  preview: EnvironmentSettings;
};

const InstancesDualInner = ({ production, preview }: InstancesDualInnerProps) => {
  const prodRegions = production.regions.map((r) => r.name);
  const previewRegions = preview.regions.map((r) => r.name);
  const defaultProdInstances = production.regions[0]?.replicas ?? 1;
  const defaultPreviewInstances = preview.regions[0]?.replicas ?? 1;

  const {
    handleSubmit,
    setValue,
    formState: { isValid, isSubmitting },
    control,
    reset,
  } = useForm<InstancesDualFormValues>({
    resolver: zodResolver(instancesDualSchema),
    mode: "onChange",
    defaultValues: {
      productionInstances: defaultProdInstances,
      previewInstances: defaultPreviewInstances,
    },
  });

  useEffect(() => {
    reset({
      productionInstances: defaultProdInstances,
      previewInstances: defaultPreviewInstances,
    });
  }, [defaultProdInstances, defaultPreviewInstances, reset]);

  const currentProdInstances = useWatch({ control, name: "productionInstances" });
  const currentPreviewInstances = useWatch({ control, name: "previewInstances" });

  const onSubmit = async (values: InstancesDualFormValues) => {
    if (values.productionInstances !== defaultProdInstances) {
      collection.environmentSettings.update(production.environmentId, (draft) => {
        for (const region of draft.regions) {
          region.replicas = values.productionInstances;
        }
      });
    }
    if (values.previewInstances !== defaultPreviewInstances) {
      collection.environmentSettings.update(preview.environmentId, (draft) => {
        for (const region of draft.regions) {
          region.replicas = values.previewInstances;
        }
      });
    }
  };

  const prodChanged = currentProdInstances !== defaultProdInstances;
  const previewChanged = currentPreviewInstances !== defaultPreviewInstances;
  const hasChanges = prodChanged || previewChanged;
  const hasProdRegions = prodRegions.length > 0;
  const hasPreviewRegions = previewRegions.length > 0;

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [
      !hasProdRegions && !hasPreviewRegions,
      { status: "disabled", reason: "Select at least one region before setting instance count" },
    ],
    [!isValid, { status: "disabled" }],
    [!hasChanges, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<Connections3 className="text-gray-12" iconSize="xl-medium" />}
      title="Instances"
      description="Number of instances running in each region"
      displayValue={
        <div className="flex items-center gap-3">
          <EnvironmentDisplayValue label="Production" instances={defaultProdInstances} />
          <span className="text-gray-8">|</span>
          <EnvironmentDisplayValue label="Preview" instances={defaultPreviewInstances} />
        </div>
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
    >
      <EnvironmentSliderSection label="Production">
        <div className="flex items-center gap-3">
          <Slider
            min={1}
            max={10}
            step={1}
            value={[currentProdInstances]}
            onValueChange={([value]) => {
              if (value !== undefined) {
                setValue("productionInstances", value, { shouldValidate: true });
              }
            }}
            className="flex-1 max-w-[480px]"
            rangeStyle={{
              background:
                "linear-gradient(to right, hsla(var(--featureA-4)), hsla(var(--featureA-12)))",
              backgroundSize: `${currentProdInstances > 1 ? 100 / ((currentProdInstances - 1) / 9) : 100}% 100%`,
              backgroundRepeat: "no-repeat",
            }}
          />
          <div className="flex items-center gap-1.5">
            {prodRegions.map((r) => (
              <RegionFlag
                key={r}
                flagCode={mapRegionToFlag(r)}
                size="xs"
                shape="circle"
                className="[&_img]:size-3"
              />
            ))}
          </div>
          <span className="text-[13px]">
            <span className="font-medium text-gray-12">{currentProdInstances}</span>{" "}
            <span className="text-gray-11 font-normal">
              instance{currentProdInstances !== 1 ? "s" : ""}
            </span>
          </span>
        </div>
      </EnvironmentSliderSection>

      <EnvironmentSliderSection label="Preview">
        <div className="flex items-center gap-3">
          <Slider
            min={1}
            max={10}
            step={1}
            value={[currentPreviewInstances]}
            onValueChange={([value]) => {
              if (value !== undefined) {
                setValue("previewInstances", value, { shouldValidate: true });
              }
            }}
            className="flex-1 max-w-[480px]"
            rangeStyle={{
              background:
                "linear-gradient(to right, hsla(var(--featureA-4)), hsla(var(--featureA-12)))",
              backgroundSize: `${currentPreviewInstances > 1 ? 100 / ((currentPreviewInstances - 1) / 9) : 100}% 100%`,
              backgroundRepeat: "no-repeat",
            }}
          />
          <div className="flex items-center gap-1.5">
            {previewRegions.map((r) => (
              <RegionFlag
                key={r}
                flagCode={mapRegionToFlag(r)}
                size="xs"
                shape="circle"
                className="[&_img]:size-3"
              />
            ))}
          </div>
          <span className="text-[13px]">
            <span className="font-medium text-gray-12">{currentPreviewInstances}</span>{" "}
            <span className="text-gray-11 font-normal">
              instance{currentPreviewInstances !== 1 ? "s" : ""}
            </span>
          </span>
        </div>
      </EnvironmentSliderSection>

      <SettingDescription>
        More instances improve availability and handle higher traffic. Changes apply on next deploy.
      </SettingDescription>
    </FormSettingCard>
  );
};

const EnvironmentDisplayValue = ({ label, instances }: { label: string; instances: number }) => (
  <div className="space-x-1">
    <span className="text-gray-11 text-xs font-normal">{label}</span>
    <span className="font-medium text-gray-12">{instances}</span>
    <span className="text-gray-11 font-normal">instance{instances !== 1 ? "s" : ""}</span>
  </div>
);
