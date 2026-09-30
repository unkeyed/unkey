"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { IconChevronDownOutline12 } from "@unkey/icons";
import {
  FormField,
  InputGroup,
  InputGroupAddon,
  InputGroupInput,
  InputGroupText,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  SettingsRow,
} from "@unkey/ui";
import { useEffect } from "react";
import { Controller, useForm } from "react-hook-form";
import { useEnvironmentSettings } from "../../../environment-provider";
import { useUpdateAllEnvironments } from "../../../hooks/use-update-all-environments";
import { SettingField } from "../../shared/form-blocks";
import { SettingsForm, resolveSaveState } from "../../shared/form-setting-card";
import { MethodBadge } from "./method-badge";
import {
  HTTP_METHODS,
  type HealthcheckFormValues,
  INTERVAL_SECONDS,
  healthcheckSchema,
} from "./schema";

export function Healthcheck() {
  const { settings, variant } = useEnvironmentSettings();
  const { healthcheck } = settings;
  const updateAllEnvironments = useUpdateAllEnvironments();

  const defaultValues: HealthcheckFormValues = {
    method: healthcheck?.method ?? "GET",
    path: healthcheck?.path ?? "",
    intervalSeconds: healthcheck?.intervalSeconds ?? INTERVAL_SECONDS.default,
  };

  const {
    handleSubmit,
    control,
    register,
    reset,
    watch,
    formState: { isValid, isSubmitting, isDirty, errors },
  } = useForm<HealthcheckFormValues>({
    resolver: zodResolver(healthcheckSchema),
    mode: "onChange",
    defaultValues,
  });

  // biome-ignore lint/correctness/useExhaustiveDependencies: reset only when the saved values change
  useEffect(() => {
    reset(defaultValues);
  }, [defaultValues.method, defaultValues.path, defaultValues.intervalSeconds, reset]);

  const onSubmit = async (values: HealthcheckFormValues) => {
    updateAllEnvironments((draft) => {
      draft.healthcheck =
        values.path === ""
          ? null
          : {
              method: values.method,
              path: values.path,
              intervalSeconds: values.intervalSeconds,
              timeoutSeconds: draft.healthcheck?.timeoutSeconds ?? 5,
              failureThreshold: draft.healthcheck?.failureThreshold ?? 3,
              initialDelaySeconds: draft.healthcheck?.initialDelaySeconds ?? 0,
            };
    });
  };

  const saveState = resolveSaveState([
    [isSubmitting, { status: "saving" }],
    [!isValid, { status: "disabled" }],
    [!isDirty, { status: "disabled", reason: "No changes to save" }],
  ]);

  const pathEmpty = watch("path") === "";

  return (
    <SettingsForm
      className="divide-y divide-grayA-4"
      onSubmit={handleSubmit(onSubmit)}
      saveState={saveState}
      autoSave={variant === "onboarding"}
    >
      <SettingsRow title="Endpoint" description="Path your app answers on">
        <SettingField>
          <FormField
            error={errors.path?.message}
            description={pathEmpty ? "No path, no health checks" : undefined}
          >
            {(field) => (
              <InputGroup variant={field.variant}>
                <Controller
                  control={control}
                  name="method"
                  render={({ field: method }) => (
                    <Select value={method.value} onValueChange={method.onChange}>
                      <SelectTrigger
                        aria-label="HTTP method"
                        variant="ghost"
                        wrapperClassName="w-auto shrink-0"
                        className="rounded-r-none border-0 border-r border-grayA-4 hover:bg-grayA-2 focus:border-grayA-4 focus:ring-0"
                        rightIcon={
                          <IconChevronDownOutline12 className="absolute right-3 text-gray-11" />
                        }
                      >
                        <SelectValue>
                          <MethodBadge method={method.value} />
                        </SelectValue>
                      </SelectTrigger>
                      <SelectContent>
                        {HTTP_METHODS.map((option) => (
                          <SelectItem key={option} value={option} className="focus:bg-gray-3">
                            <MethodBadge method={option} />
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  )}
                />
                <InputGroupInput
                  id={field.id}
                  aria-label="Path"
                  aria-invalid={field.invalid}
                  aria-describedby={field.describedBy}
                  placeholder="/health"
                  autoComplete="off"
                  spellCheck={false}
                  className="px-3 font-mono"
                  {...register("path")}
                />
              </InputGroup>
            )}
          </FormField>
        </SettingField>
      </SettingsRow>
      <SettingsRow title="Interval" description="How often Unkey checks">
        <SettingField>
          <FormField error={errors.intervalSeconds?.message}>
            {(field) => (
              <InputGroup variant={field.variant} className="w-28">
                <InputGroupInput
                  id={field.id}
                  type="number"
                  inputMode="numeric"
                  min={INTERVAL_SECONDS.min}
                  max={INTERVAL_SECONDS.max}
                  step={1}
                  aria-label="Interval in seconds"
                  aria-invalid={field.invalid}
                  aria-describedby={field.describedBy}
                  className="pl-3 font-mono tabular-nums"
                  onWheelCapture={(e) => e.currentTarget.blur()}
                  {...register("intervalSeconds", { valueAsNumber: true })}
                />
                <InputGroupAddon align="inline-end">
                  <InputGroupText className="font-mono">s</InputGroupText>
                </InputGroupAddon>
              </InputGroup>
            )}
          </FormField>
        </SettingField>
      </SettingsRow>
    </SettingsForm>
  );
}
