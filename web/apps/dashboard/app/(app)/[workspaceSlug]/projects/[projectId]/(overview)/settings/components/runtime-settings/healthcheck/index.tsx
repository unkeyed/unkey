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
import { Controller, useForm } from "react-hook-form";
import { useEnvironmentSettings } from "../../../environment-provider";
import { FormSettingCard, resolveSaveState } from "../../shared/form-setting-card";
import { RemoveButton } from "../../shared/remove-button";
import { MethodBadge } from "./method-badge";
import { HTTP_METHODS, type HealthcheckFormValues, healthcheckSchema } from "./schema";
import { intervalToSeconds, secondsToInterval } from "./utils";

export const Healthcheck = () => {
  const { settings, variant } = useEnvironmentSettings();
  const { healthcheck, environmentId } = settings;

  const defaultValues: HealthcheckFormValues = {
    method: healthcheck?.method ?? "GET",
    path: healthcheck?.path ?? "",
    interval: healthcheck ? secondsToInterval(healthcheck.intervalSeconds) : "",
  };

  const {
    handleSubmit,
    control,
    register,
    reset,
    formState: { isValid, isSubmitting, isDirty, errors },
  } = useForm<HealthcheckFormValues>({
    resolver: zodResolver(healthcheckSchema),
    mode: "onChange",
    defaultValues,
  });

  // biome-ignore lint/correctness/useExhaustiveDependencies: we gucci
  useEffect(() => {
    reset(defaultValues);
  }, [defaultValues.method, defaultValues.path, defaultValues.interval, reset]);

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

  const handleRemove = () => {
    collection.environmentSettings.update(environmentId, (draft) => {
      draft.healthcheck = null;
    });
    reset({ method: "GET", path: "", interval: "" });
  };

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!isDirty, { status: "disabled", reason: "No changes to save" }],
  ]);

  return (
    <FormSettingCard
      icon={<HeartPulse className="text-gray-12" iconSize="xl-medium" />}
      title="Healthcheck"
      description="Endpoint used to verify the service is healthy"
      displayValue={
        healthcheck ? (
          <div className="flex gap-1.5 items-center justify-center">
            <MethodBadge method={healthcheck.method} />
            <span className="font-medium text-gray-12">{healthcheck.path}</span>
            <span className="text-gray-11 font-normal">every {healthcheck.intervalSeconds}s</span>
          </div>
        ) : null
      }
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <div className="flex flex-col gap-2 w-120">
        {/* TODO: multi-check when API supports
        {fields.map((field, index) => (
          <div key={field.id} className="flex items-end gap-3">
            ... add/remove buttons and per-entry fields ...
          </div>
        ))}
        */}
        <div className="flex items-center gap-3">
          <span className="w-24 text-[13px] text-gray-11">Method</span>
          <span className="flex-1 text-[13px] text-gray-11">Path</span>
          <span className="flex-1 text-[13px] text-gray-11">Interval</span>
        </div>
        <div className="relative flex items-start gap-3">
          <Controller
            control={control}
            name="method"
            render={({ field }) => (
              <Select value={field.value} onValueChange={field.onChange}>
                <SelectTrigger
                  className="h-9"
                  wrapperClassName="w-24"
                  variant={errors.method ? "error" : "default"}
                  rightIcon={
                    <ChevronDown
                      className="absolute right-3 size-3 text-gray-11"
                      iconSize="sm-medium"
                    />
                  }
                >
                  <SelectValue placeholder={<MethodBadge method={"GET"} />}>
                    <MethodBadge method={field.value} />
                  </SelectValue>
                </SelectTrigger>
                <SelectContent>
                  {HTTP_METHODS.map((method) => (
                    <SelectItem key={method} value={method} className="focus:bg-gray-3">
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
          {healthcheck && (
            <RemoveButton onClick={handleRemove} className="absolute -right-11 top-0" />
          )}
        </div>
      </div>
    </FormSettingCard>
  );
};
