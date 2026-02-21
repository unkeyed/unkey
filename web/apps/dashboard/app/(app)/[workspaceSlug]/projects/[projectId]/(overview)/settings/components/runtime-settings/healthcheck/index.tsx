"use client";

import { collection } from "@/lib/collections";
import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronDown, HeartPulse } from "@unkey/icons";
import {
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@unkey/ui";
import { useEffect } from "react";
import { Controller, useForm, useWatch } from "react-hook-form";
import { useEnvironmentSettings } from "../../../environment-provider";
import { FormSettingCard } from "../../shared/form-setting-card";
import { MethodBadge } from "./method-badge";
import { HTTP_METHODS, type HealthcheckFormValues, healthcheckSchema } from "./schema";
import { intervalToSeconds, secondsToInterval } from "./utils";

export const Healthcheck = () => {
  const { settings } = useEnvironmentSettings();
  const { healthcheck, environmentId } = settings;

  const defaultValues: HealthcheckFormValues = {
    method: healthcheck?.method ?? "GET",
    path: healthcheck?.path ?? "/health",
    interval: healthcheck ? secondsToInterval(healthcheck.intervalSeconds) : "30s",
  };

  const {
    handleSubmit,
    control,
    register,
    reset,
    formState: { isValid, isSubmitting, errors },
  } = useForm<HealthcheckFormValues>({
    resolver: zodResolver(healthcheckSchema),
    mode: "onChange",
    defaultValues,
  });

  // biome-ignore lint/correctness/useExhaustiveDependencies: we gucci
  useEffect(() => {
    reset(defaultValues);
  }, [defaultValues.method, defaultValues.path, defaultValues.interval, reset]);

  const currentMethod = useWatch({ control, name: "method" });
  const currentPath = useWatch({ control, name: "path" });
  const currentInterval = useWatch({ control, name: "interval" });

  const onSubmit = async (values: HealthcheckFormValues) => {
    collection.environmentSettings.update(environmentId, (draft) => {
      draft.healthcheck =
        values.path.trim() === ""
          ? null
          : {
              method: values.method,
              path: values.path.trim(),
              intervalSeconds: intervalToSeconds(values.interval),
              timeoutSeconds: 5,
              failureThreshold: 3,
              initialDelaySeconds: 0,
            };
    });
  };

  const hasChanges =
    currentMethod !== defaultValues.method ||
    currentPath !== defaultValues.path ||
    currentInterval !== defaultValues.interval;

  return (
    <FormSettingCard
      icon={<HeartPulse className="text-gray-12" iconSize="xl-medium" />}
      title="Healthcheck"
      description="Endpoint used to verify the service is healthy"
      displayValue={
        <div className="flex gap-1.5 items-center justify-center">
          <MethodBadge method={defaultValues.method} />
          <span className="font-medium text-gray-12">{defaultValues.path}</span>
          <span className="text-gray-11 font-normal">every {defaultValues.interval}</span>
        </div>
      }
      onSubmit={handleSubmit(onSubmit)}
      canSave={isValid && !isSubmitting && hasChanges}
      isSaving={isSubmitting}
    >
      <div className="flex flex-col gap-3 w-[520px]">
        {/* TODO: multi-check when API supports
        {fields.map((field, index) => (
          <div key={field.id} className="flex items-end gap-3">
            ... add/remove buttons and per-entry fields ...
          </div>
        ))}
        */}
        <div className="flex items-center gap-3">
          <span className="w-20 text-[13px] text-gray-11">Method</span>
          <span className="flex-1 text-[13px] text-gray-11">Path</span>
          <span className="flex-1 text-[13px] text-gray-11">Interval</span>
        </div>
        <div className="flex items-start gap-3">
          <Controller
            control={control}
            name="method"
            render={({ field }) => (
              <Select value={field.value} onValueChange={field.onChange}>
                <SelectTrigger
                  className="h-9"
                  wrapperClassName="w-20"
                  variant={errors.method ? "error" : "default"}
                  rightIcon={<ChevronDown className="absolute right-3 size-3 opacity-70" />}
                >
                  <SelectValue>
                    <MethodBadge method={field.value} />
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {HTTP_METHODS.map((method) => (
                    <SelectItem key={method} value={method}>
                      <MethodBadge method={method} />
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          />
          <FormInput
            placeholder="/health"
            className="flex-1 [&_input]:h-9 [&_input]:font-mono"
            error={errors.path?.message}
            {...register("path")}
          />
          <FormInput
            className="flex-1 [&_input]:h-9"
            placeholder="30s"
            error={errors.interval?.message}
            {...register("interval")}
          />
        </div>
      </div>
    </FormSettingCard>
  );
};
