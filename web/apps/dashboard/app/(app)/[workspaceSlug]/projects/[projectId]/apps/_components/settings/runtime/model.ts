import {
  ENVIRONMENT_SETTINGS_DEFAULTS,
  type EnvironmentSettings,
} from "@/lib/collections/deploy/environment-settings";
import { sameJson } from "@/lib/utils/same-json";
import { EnvironmentShutdownSignal } from "@unkey/api/models/components";
import { z } from "zod";
import {
  type ComputeValues,
  computeSchema,
  readCompute,
  replicasError,
  writeCompute,
} from "../compute/draft";

export const HTTP_METHODS = ["GET", "POST"] as const;

export const HEALTHCHECK_FIELDS = [
  {
    name: "intervalSeconds",
    label: "Interval",
    ariaLabel: "Interval in seconds",
    min: 1,
    max: 3600,
    suffix: "s",
    noun: "seconds",
  },
  {
    name: "timeoutSeconds",
    label: "Timeout",
    ariaLabel: "Timeout in seconds",
    min: 1,
    max: 3600,
    suffix: "s",
    noun: "seconds",
  },
  {
    name: "failureThreshold",
    label: "Failure threshold",
    ariaLabel: "Failure threshold",
    min: 1,
    max: 100,
    suffix: null,
    noun: "failures",
  },
  {
    name: "initialDelaySeconds",
    label: "Initial delay",
    ariaLabel: "Initial delay in seconds",
    min: 0,
    max: 3600,
    suffix: "s",
    noun: "seconds",
  },
] as const;

const PORT_MESSAGE = "Use a port from 1 to 65535";

const PATH_PATTERN = /^(\/[\w-]+)+(\.\w+)?$/;

const timingSchema = z.number().or(z.nan());

const healthcheckSchema = z
  .object({
    method: z.enum(HTTP_METHODS),
    path: z
      .string()
      .refine((path) => path === "" || path.startsWith("/"), "Path must start with /")
      .refine(
        (path) => path === "" || PATH_PATTERN.test(path),
        "Start with / and use letters, numbers, - and _, like /health",
      ),
    intervalSeconds: timingSchema,
    timeoutSeconds: timingSchema,
    failureThreshold: timingSchema,
    initialDelaySeconds: timingSchema,
  })
  .superRefine((values, ctx) => {
    if (values.path === "") {
      return;
    }
    for (const { name, min, max, noun } of HEALTHCHECK_FIELDS) {
      const value = values[name];
      if (!Number.isInteger(value) || value < min || value > max) {
        ctx.addIssue({
          code: "custom",
          path: [name],
          message: `Use ${min} to ${max} ${noun}`,
        });
      }
    }
  });

export const runtimeSchema = z.object({
  compute: computeSchema,
  port: z
    .number({ error: PORT_MESSAGE })
    .int(PORT_MESSAGE)
    .min(1, PORT_MESSAGE)
    .max(65535, PORT_MESSAGE),
  command: z.string().trim(),
  shutdownSignal: z.enum(EnvironmentShutdownSignal),
  healthcheck: healthcheckSchema,
});

export type RuntimeValues = z.infer<typeof runtimeSchema>;

type HealthcheckValues = RuntimeValues["healthcheck"];

export type RuntimeSection = keyof RuntimeValues;

const UNCHECKED: { [K in RuntimeSection]: z.ZodType<RuntimeValues[K], RuntimeValues[K]> } = {
  compute: z.custom<ComputeValues>(),
  port: z.custom<number>(),
  command: z.custom<string>(),
  shutdownSignal: z.custom<RuntimeValues["shutdownSignal"]>(),
  healthcheck: z.custom<HealthcheckValues>(),
};

export function runtimeFormSchema(
  shown: readonly RuntimeSection[] | "all",
  replicaLimit: number,
): z.ZodType<RuntimeValues, RuntimeValues> {
  const checks = {
    ...runtimeSchema.shape,
    compute: computeSchema.superRefine(({ replicas }, ctx) => {
      const message = replicasError(replicas, replicaLimit);
      if (message) {
        ctx.addIssue({ code: "custom", path: ["replicas"], message });
      }
    }),
  };
  const section = <K extends RuntimeSection>(key: K) =>
    shown === "all" || shown.includes(key) ? checks[key] : UNCHECKED[key];
  return z.object({
    compute: section("compute"),
    port: section("port"),
    command: section("command"),
    shutdownSignal: section("shutdownSignal"),
    healthcheck: section("healthcheck"),
  });
}

const DEFAULT_HEALTHCHECK: HealthcheckValues = Object.freeze({
  ...ENVIRONMENT_SETTINGS_DEFAULTS.healthcheck,
  method: "GET",
  path: "",
});

export const TIMING_PATHS = HEALTHCHECK_FIELDS.map(({ name }) => `healthcheck.${name}` as const);

export const ADVANCED_PATHS = ["shutdownSignal", ...TIMING_PATHS] as const;

export function hasAdvancedOverrides(values: RuntimeValues): boolean {
  return (
    values.shutdownSignal !== ENVIRONMENT_SETTINGS_DEFAULTS.shutdownSignal ||
    HEALTHCHECK_FIELDS.some(
      ({ name }) => values.healthcheck[name] !== ENVIRONMENT_SETTINGS_DEFAULTS.healthcheck[name],
    )
  );
}

export function readRuntime(settings: EnvironmentSettings): RuntimeValues {
  return {
    compute: readCompute(settings),
    port: settings.port,
    command: settings.command.join(" "),
    shutdownSignal: settings.shutdownSignal,
    healthcheck: settings.healthcheck ?? DEFAULT_HEALTHCHECK,
  };
}

/**
 * Writes only the sections that changed. The command round trip through one
 * line is lossy, so an untouched command must keep its stored arguments.
 */
export function writeRuntime(
  draft: EnvironmentSettings,
  values: RuntimeValues,
  saved: RuntimeValues,
): void {
  if (!sameJson(values.compute, saved.compute)) {
    writeCompute(draft, values.compute, saved.compute);
  }
  if (values.port !== saved.port) {
    draft.port = values.port;
  }
  if (values.command !== saved.command) {
    writeCommand(draft, values.command);
  }
  if (values.shutdownSignal !== saved.shutdownSignal) {
    draft.shutdownSignal = values.shutdownSignal;
  }
  if (!sameJson(values.healthcheck, saved.healthcheck)) {
    writeHealthcheck(draft, values.healthcheck, saved.healthcheck);
  }
}

function writeCommand(draft: EnvironmentSettings, command: string): void {
  draft.command = command === "" ? [] : command.split(/\s+/);
}

function writeHealthcheck(
  draft: EnvironmentSettings,
  values: HealthcheckValues,
  saved: HealthcheckValues,
): void {
  if (values.path === "") {
    draft.healthcheck = null;
    return;
  }
  const kept = draft.healthcheck ?? values;
  const pick = <K extends keyof HealthcheckValues>(key: K) =>
    values[key] === saved[key] ? kept[key] : values[key];
  draft.healthcheck = {
    method: pick("method"),
    path: pick("path"),
    intervalSeconds: pick("intervalSeconds"),
    timeoutSeconds: pick("timeoutSeconds"),
    failureThreshold: pick("failureThreshold"),
    initialDelaySeconds: pick("initialDelaySeconds"),
  };
}
