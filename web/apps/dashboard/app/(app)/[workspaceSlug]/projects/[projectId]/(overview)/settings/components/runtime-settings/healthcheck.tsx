"use client";

import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronDown, Heart, Plus, Trash } from "@unkey/icons";
import { Badge, FormDescription, FormInput, Select, SelectContent, SelectItem, SelectTrigger, SelectValue } from "@unkey/ui";
import { type FieldErrors, Controller, useFieldArray, useForm } from "react-hook-form";
import { z } from "zod";
import { EditableSettingCard } from "../shared/editable-setting-card";

const HTTP_METHODS = ["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"] as const;

const INTERVAL_REGEX = /^\d+[smh]$/;

const healthcheckEntrySchema = z.object({
  method: z.string().min(1, "Required"),
  path: z.string()
    .min(1, "Path is required")
    .startsWith("/", "Path must start with /")
    .regex(/^\/[\w\-./]*$/, "Invalid path characters"),
  interval: z.string()
    .min(1, "Interval is required")
    .regex(INTERVAL_REGEX, "Use format like 15s, 2m, or 1h"),
});

const MAX_CHECKS = 3;

const healthcheckSchema = z.object({
  checks: z.array(healthcheckEntrySchema).min(1).max(MAX_CHECKS),
});

type HealthcheckFormValues = z.infer<typeof healthcheckSchema>;

const DEFAULT_CHECKS: HealthcheckFormValues["checks"] = [
  { method: "GET", path: "/health", interval: "30s" },
];

export const Healthcheck = () => {
  const {
    register,
    control,
    formState: { errors },
  } = useForm<HealthcheckFormValues>({
    resolver: zodResolver(healthcheckSchema),
    mode: "onChange",
    defaultValues: { checks: DEFAULT_CHECKS },
  });

  const { fields, append, remove } = useFieldArray({ control, name: "checks" });

  return (
    <EditableSettingCard
      icon={<Heart className="text-gray-12" iconSize="xl-medium" />}
      title="Healthcheck"
      description="Endpoint used to verify the service is healthy"
      border="bottom"
      displayValue={
        <div className="flex gap-1.5 items-center justify-center">
          <MethodBadge method="GET" />
          <span className="font-medium text-gray-12">/health</span>
          <span className="text-gray-11 font-normal">every 30s</span>
        </div>
      }
      formId="update-healthcheck-form"
      canSave={false}
      isSaving={false}
    >
      <form id="update-healthcheck-form">
        <div className="flex flex-col gap-3 w-[480px]">
          {fields.map((field, index) => (
            <div key={field.id} className="flex items-end gap-3">
              <div className="flex flex-col">
                {index === 0 && (
                  <label className="text-gray-11 text-[13px] leading-5 mb-1.5">Method</label>
                )}
                <Controller
                  control={control}
                  name={`checks.${index}.method`}
                  render={({ field }) => (
                    <Select value={field.value} onValueChange={field.onChange}>
                      <SelectTrigger
                        className="h-9"
                        variant={errors.checks?.[index]?.method ? "error" : "default"}
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
              </div>
              <FormInput
                label={index === 0 ? "Path" : undefined}
                placeholder="/health"
                className="flex-1 [&_input]:h-9"
                variant={errors.checks?.[index]?.path ? "error" : "default"}
                {...register(`checks.${index}.path`)}
              />
              <FormInput
                label={index === 0 ? "Interval" : undefined}
                className="[&_input]:h-9"
                placeholder="30s"
                variant={errors.checks?.[index]?.interval ? "error" : "default"}
                {...register(`checks.${index}.interval`)}
              />
              <div className="flex gap-1 h-9 items-center w-[60px] shrink-0 justify-start">
                {fields.length > 1 && (
                  <button
                    type="button"
                    onClick={() => remove(index)}
                    className="p-1.5 rounded-md text-gray-11 hover:text-gray-12 hover:bg-gray-3 transition-colors"
                  >
                    <Trash iconSize="sm-regular" />
                  </button>
                )}
                {index === fields.length - 1 && fields.length < MAX_CHECKS ? (
                  <button
                    type="button"
                    onClick={() => append({ method: "GET", path: "/health", interval: "30s" })}
                    className="p-1.5 rounded-md text-gray-11 hover:text-gray-12 hover:bg-gray-3 transition-colors"
                  >
                    <Plus iconSize="sm-regular" />
                  </button>
                ) : (
                  <div className="w-[30px] shrink-0" />
                )}
              </div>
            </div>
          ))}
        </div>
        <div className="mt-1">
          <FormDescription
            description="Defines the endpoint and frequency used to check if your service is running. Changes apply on next deploy."
            error={getFirstCheckError(errors)}
            descriptionId="healthcheck-description"
            errorId="healthcheck-error"
          />
        </div>
      </form>
    </EditableSettingCard>
  );
};

function getMethodVariant(method: string): "success" | "warning" | "error" | "primary" | "blocked" {
  switch (method) {
    case "GET":
    case "HEAD":
      return "success";
    case "POST":
      return "warning";
    case "PUT":
    case "PATCH":
      return "blocked";
    case "DELETE":
      return "error";
    default:
      return "primary";
  }
}

function getFirstCheckError(errors: FieldErrors<HealthcheckFormValues>): string | undefined {
  const checks = errors.checks;
  if (!checks) {
    return undefined;
  }
  if (Array.isArray(checks)) {
    for (const entry of checks) {
      if (!entry) {
        continue;
      }
      for (const field of ["method", "path", "interval"] as const) {
        if (entry[field]?.message) {
          return entry[field].message;
        }
      }
    }
  }
  if (!Array.isArray(checks) && checks.root?.message) {
    return checks.root.message;
  }
  return undefined;
}

const MethodBadge: React.FC<{ method: string }> = ({ method }) => (
  <Badge
    variant={getMethodVariant(method)}
    size="sm"
    className="text-[11px] font-medium w-7 h-[18px] flex items-center justify-center"
  >
    {method}
  </Badge>
);
