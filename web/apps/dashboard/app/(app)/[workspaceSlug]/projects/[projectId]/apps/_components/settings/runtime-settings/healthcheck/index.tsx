"use client";

import { ENVIRONMENT_SETTINGS_DEFAULTS } from "@/lib/collections/deploy/environment-settings";
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
} from "@unkey/ui";
import { Controller } from "react-hook-form";
import { useSettingForm } from "../../hooks/use-setting-form";
import { MethodBadge } from "./method-badge";
import {
  HTTP_METHODS,
  type HealthcheckFormValues,
  INTERVAL_SECONDS,
  healthcheckSchema,
} from "./schema";

export function Healthcheck() {
  const { form, formProps } = useSettingForm({
    schema: healthcheckSchema,
    read: ({ healthcheck }): HealthcheckFormValues => ({
      method: healthcheck?.method ?? "GET",
      path: healthcheck?.path ?? "",
      intervalSeconds: healthcheck?.intervalSeconds ?? INTERVAL_SECONDS.default,
    }),
    write: (draft, values) => {
      draft.healthcheck =
        values.path === ""
          ? null
          : {
              ...ENVIRONMENT_SETTINGS_DEFAULTS.healthcheck,
              ...draft.healthcheck,
              method: values.method,
              path: values.path,
              intervalSeconds: values.intervalSeconds,
            };
    },
  });
  const { control, register } = form;
  const { errors } = form.formState;
  const pathEmpty = form.watch("path") === "";

  return (
    <SettingsForm className="divide-y divide-grayA-4" {...formProps}>
      <SettingsRow title="Endpoint">
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
      </SettingsRow>
      <SettingsRow title="Interval">
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
      </SettingsRow>
    </SettingsForm>
  );
}
