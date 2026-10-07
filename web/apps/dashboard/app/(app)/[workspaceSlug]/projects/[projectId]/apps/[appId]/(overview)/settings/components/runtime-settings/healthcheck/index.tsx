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
  SettingsForm,
  SettingsRow,
  SettingsRowContent,
  SettingsRowHeader,
  SettingsRowTitle,
  formSaveState,
} from "@unkey/ui";
import { Controller, useForm } from "react-hook-form";
import { useEnvironmentSettings } from "../../../environment-provider";
import { useUpdateAllEnvironments } from "../../../hooks/use-update-all-environments";
import { SettingField } from "../../shared/form-blocks";
import { MethodBadge } from "./method-badge";
import {
  HTTP_METHODS,
  type HealthcheckFormValues,
  INTERVAL_SECONDS,
  healthcheckSchema,
} from "./schema";

export function Healthcheck() {
  const { settings, autoSave } = useEnvironmentSettings();
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
    watch,
    formState: { isValid, isSubmitting, isDirty, errors },
  } = useForm<HealthcheckFormValues>({
    resolver: zodResolver(healthcheckSchema),
    mode: "onChange",
    values: defaultValues,
  });

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

  const saveState = formSaveState({
    isSubmitting,
    isValid,
    isDirty: isDirty,
  });

  const pathEmpty = watch("path") === "";

  return (
    <SettingsForm
      className="divide-y divide-grayA-4"
      onSubmit={handleSubmit(onSubmit)}
      dirty={isDirty}
      saveState={saveState}
      autoSave={autoSave}
    >
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>Endpoint</SettingsRowTitle>
        </SettingsRowHeader>
        <SettingsRowContent>
          <SettingField>
            <FormField
              error={errors.path?.message}
              description={pathEmpty ? "Leave empty to turn off health checks." : undefined}
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
                    data-1p-ignore
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
        </SettingsRowContent>
      </SettingsRow>
      <SettingsRow>
        <SettingsRowHeader>
          <SettingsRowTitle>Interval</SettingsRowTitle>
        </SettingsRowHeader>
        <SettingsRowContent>
          <SettingField>
            <FormField error={errors.intervalSeconds?.message}>
              {(field) => (
                <InputGroup variant={field.variant} className="w-28">
                  <InputGroupInput
                    data-1p-ignore
                    id={field.id}
                    type="number"
                    inputMode="numeric"
                    min={INTERVAL_SECONDS.min}
                    max={INTERVAL_SECONDS.max}
                    step={1}
                    aria-label="Interval in seconds"
                    aria-invalid={field.invalid}
                    aria-describedby={field.describedBy}
                    disabled={pathEmpty}
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
        </SettingsRowContent>
      </SettingsRow>
    </SettingsForm>
  );
}
