"use client";

import { trpc } from "@/lib/trpc/client";
import { zodResolver } from "@hookform/resolvers/zod";
import { ChevronDown, Heart } from "@unkey/icons";
import {
  Badge,
  FormDescription,
  FormInput,
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
  toast,
} from "@unkey/ui";
import { useEffect } from "react";
import { Controller, useForm, useWatch } from "react-hook-form";
import { z } from "zod";
import { useProjectData } from "../../../data-provider";
import { FormSettingCard } from "../shared/form-setting-card";

// TODO: extend when API supports more methods
const HTTP_METHODS = ["GET", "POST"] as const;

const INTERVAL_REGEX = /^\d+[smh]$/;

// TODO: MAX_CHECKS = 3 and array schema for multi-check when API supports
const healthcheckSchema = z.object({
  method: z.enum(["GET", "POST"]),
  path: z
    .string()
    .min(1, "Path is required")
    .startsWith("/", "Path must start with /")
    .regex(/^\/[\w\-./]*$/, "Invalid path characters"),
  interval: z
    .string()
    .min(1, "Interval is required")
    .regex(INTERVAL_REGEX, "Use format like 15s, 2m, or 1h"),
});

type HealthcheckFormValues = z.infer<typeof healthcheckSchema>;

function intervalToSeconds(interval: string): number {
  const num = Number.parseInt(interval, 10);
  if (interval.endsWith("h")) return num * 3600;
  if (interval.endsWith("m")) return num * 60;
  return num;
}

function secondsToInterval(seconds: number): string {
  if (seconds % 3600 === 0) return `${seconds / 3600}h`;
  if (seconds % 60 === 0) return `${seconds / 60}m`;
  return `${seconds}s`;
}

export const Healthcheck = () => {
  const { environments } = useProjectData();
  const environmentId = environments[0]?.id;

  const { data: settingsData } = trpc.deploy.environmentSettings.get.useQuery(
    { environmentId: environmentId ?? "" },
    { enabled: Boolean(environmentId) },
  );

  const healthcheck = settingsData?.runtimeSettings?.healthcheck;
  const defaultValues: HealthcheckFormValues = {
    method: healthcheck?.method ?? "GET",
    path: healthcheck?.path ?? "/health",
    interval: healthcheck ? secondsToInterval(healthcheck.intervalSeconds) : "30s",
  };

  return <HealthcheckForm environmentId={environmentId ?? ""} defaultValues={defaultValues} />;
};

type HealthcheckFormProps = {
  environmentId: string;
  defaultValues: HealthcheckFormValues;
};

const HealthcheckForm: React.FC<HealthcheckFormProps> = ({ environmentId, defaultValues }) => {
  const utils = trpc.useUtils();

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

  useEffect(() => {
    reset(defaultValues);
  }, [defaultValues.method, defaultValues.path, defaultValues.interval, reset]);

  const currentMethod = useWatch({ control, name: "method" });
  const currentPath = useWatch({ control, name: "path" });
  const currentInterval = useWatch({ control, name: "interval" });

  const updateRuntime = trpc.deploy.environmentSettings.updateRuntime.useMutation({
    onSuccess: () => {
      toast.success("Healthcheck updated", { duration: 5000 });
      utils.deploy.environmentSettings.get.invalidate({ environmentId });
    },
    onError: (err) => {
      if (err.data?.code === "BAD_REQUEST") {
        toast.error("Invalid healthcheck setting", {
          description: err.message || "Please check your input and try again.",
        });
      } else {
        toast.error("Failed to update healthcheck", {
          description:
            err.message ||
            "An unexpected error occurred. Please try again or contact support@unkey.com",
          action: {
            label: "Contact Support",
            onClick: () => window.open("mailto:support@unkey.com", "_blank"),
          },
        });
      }
    },
  });

  const onSubmit = async (values: HealthcheckFormValues) => {
    await updateRuntime.mutateAsync({
      environmentId,
      healthcheck:
        values.path.trim() === ""
          ? null
          : {
              method: values.method,
              path: values.path.trim(),
              intervalSeconds: intervalToSeconds(values.interval),
              timeoutSeconds: 5,
              failureThreshold: 3,
              initialDelaySeconds: 0,
            },
    });
  };

  const hasChanges =
    currentMethod !== defaultValues.method ||
    currentPath !== defaultValues.path ||
    currentInterval !== defaultValues.interval;

  return (
    <FormSettingCard
      icon={<Heart className="text-gray-12" iconSize="xl-medium" />}
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
      isSaving={updateRuntime.isLoading || isSubmitting}
    >
      <div className="flex flex-col gap-3 w-[480px]">
        {/* TODO: multi-check when API supports
        {fields.map((field, index) => (
          <div key={field.id} className="flex items-end gap-3">
            ... add/remove buttons and per-entry fields ...
          </div>
        ))}
        */}
        <div className="flex items-end gap-3">
          <div className="flex flex-col">
            <label className="text-gray-11 text-[13px] leading-5 mb-1.5">Method</label>
            <Controller
              control={control}
              name="method"
              render={({ field }) => (
                <Select value={field.value} onValueChange={field.onChange}>
                  <SelectTrigger
                    className="h-9"
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
          </div>
          <FormInput
            label="Path"
            placeholder="/health"
            className="flex-1 [&_input]:h-9"
            variant={errors.path ? "error" : "default"}
            {...register("path")}
          />
          <FormInput
            label="Interval"
            className="[&_input]:h-9"
            placeholder="30s"
            variant={errors.interval ? "error" : "default"}
            {...register("interval")}
          />
        </div>
      </div>
      <div className="mt-1">
        <FormDescription
          description="Defines the endpoint and frequency used to check if your service is running. Changes apply on next deploy."
          error={errors.method?.message ?? errors.path?.message ?? errors.interval?.message}
          descriptionId="healthcheck-description"
          errorId="healthcheck-error"
        />
      </div>
    </FormSettingCard>
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

const MethodBadge: React.FC<{ method: string }> = ({ method }) => (
  <Badge
    variant={getMethodVariant(method)}
    size="sm"
    className="text-[11px] font-medium w-7 h-[18px] flex items-center justify-center"
  >
    {method}
  </Badge>
);
