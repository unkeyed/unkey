"use client";

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
  SettingsRowContent,
  SettingsRowHeader,
  SettingsRowTitle,
} from "@unkey/ui";
import { Controller, type UseFormReturn } from "react-hook-form";
import { MethodBadge } from "./method-badge";
import { HEALTHCHECK_FIELDS, HTTP_METHODS, type RuntimeValues, TIMING_PATHS } from "./model";

type HealthcheckProps = { form: UseFormReturn<RuntimeValues> };

export function HealthcheckFields({ form }: HealthcheckProps) {
  const { control, register } = form;
  const errors = form.formState.errors.healthcheck;
  const pathEmpty = form.watch("healthcheck.path") === "";

  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>Health check path</SettingsRowTitle>
      </SettingsRowHeader>
      <SettingsRowContent>
        <FormField
          className="max-w-(--setting-w)"
          error={errors?.path?.message}
          description={pathEmpty ? "Leave empty to turn off health checks." : undefined}
        >
          {(field) => (
            <InputGroup variant={field.variant}>
              <Controller
                control={control}
                name="healthcheck.method"
                render={({ field: method }) => (
                  <Select value={method.value} onValueChange={method.onChange} disabled={pathEmpty}>
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
                aria-label="Health check path"
                aria-invalid={field.invalid}
                aria-describedby={field.describedBy}
                placeholder="/health"
                autoComplete="off"
                spellCheck={false}
                className="px-3 font-mono"
                {...register("healthcheck.path", { deps: TIMING_PATHS })}
              />
            </InputGroup>
          )}
        </FormField>
      </SettingsRowContent>
    </SettingsRow>
  );
}

export function HealthcheckTiming({ form }: HealthcheckProps) {
  const { register } = form;
  const errors = form.formState.errors.healthcheck;
  const pathEmpty = form.watch("healthcheck.path") === "";

  return (
    <SettingsRow>
      <SettingsRowHeader>
        <SettingsRowTitle>Health check timing</SettingsRowTitle>
      </SettingsRowHeader>
      <SettingsRowContent>
        <div className="grid grid-cols-2 items-start gap-3">
          {HEALTHCHECK_FIELDS.map(({ name, label, ariaLabel, min, max, suffix }) => (
            <FormField key={name} label={label} className="min-w-0" error={errors?.[name]?.message}>
              {(field) => (
                <InputGroup variant={field.variant}>
                  <InputGroupInput
                    data-1p-ignore
                    id={field.id}
                    type="number"
                    inputMode="numeric"
                    min={min}
                    max={max}
                    step={1}
                    aria-label={ariaLabel}
                    aria-invalid={field.invalid}
                    aria-describedby={field.describedBy}
                    disabled={pathEmpty}
                    className="pl-3 font-mono tabular-nums"
                    onWheelCapture={(e) => e.currentTarget.blur()}
                    {...register(`healthcheck.${name}`, { valueAsNumber: true })}
                  />
                  {suffix ? (
                    <InputGroupAddon align="inline-end">
                      <InputGroupText className="font-mono">{suffix}</InputGroupText>
                    </InputGroupAddon>
                  ) : null}
                </InputGroup>
              )}
            </FormField>
          ))}
        </div>
      </SettingsRowContent>
    </SettingsRow>
  );
}
