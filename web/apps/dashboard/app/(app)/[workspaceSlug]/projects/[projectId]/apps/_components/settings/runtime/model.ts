import {
  ENVIRONMENT_SETTINGS_DEFAULTS,
  type EnvironmentSettings,
} from "@/lib/collections/deploy/environment-settings";
import { sameJson } from "@/lib/utils/same-json";
import { z } from "zod";
import { type ComputeValues, computeSchema, readCompute, writeCompute } from "../compute/draft";

export const HTTP_METHODS = ["GET", "POST"] as const;

export const INTERVAL_SECONDS = {
  min: 1,
  max: 3600,
  default: ENVIRONMENT_SETTINGS_DEFAULTS.healthcheck.intervalSeconds,
} as const;

const PORT_MESSAGE = "Use a port from 1 to 65535";

const INTERVAL_MESSAGE = `Use ${INTERVAL_SECONDS.min} to ${INTERVAL_SECONDS.max} seconds`;

const PATH_PATTERN = /^(\/[\w-]+)+(\.\w+)?$/;

const intervalSchema = z.number().int().min(INTERVAL_SECONDS.min).max(INTERVAL_SECONDS.max);

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
    intervalSeconds: z.number().or(z.nan()),
  })
  .superRefine(({ path, intervalSeconds }, ctx) => {
    if (path !== "" && !intervalSchema.safeParse(intervalSeconds).success) {
      ctx.addIssue({ code: "custom", path: ["intervalSeconds"], message: INTERVAL_MESSAGE });
    }
  });

export const runtimeSchema = z.object({
  compute: computeSchema,
  port: z
    .number({ error: PORT_MESSAGE })
    .int(PORT_MESSAGE)
    .min(1, PORT_MESSAGE)
    .max(65535, PORT_MESSAGE),
  command: z.string(),
  healthcheck: healthcheckSchema,
});

export type RuntimeValues = z.infer<typeof runtimeSchema>;

type HealthcheckValues = RuntimeValues["healthcheck"];

export const computeSectionSchema = runtimeSchema.extend({
  port: z.custom<number>(),
  command: z.custom<string>(),
  healthcheck: z.custom<HealthcheckValues>(),
});

export const commandSectionSchema = runtimeSchema.extend({
  compute: z.custom<ComputeValues>(),
  port: z.custom<number>(),
  healthcheck: z.custom<HealthcheckValues>(),
});

export function readRuntime(settings: EnvironmentSettings): RuntimeValues {
  return {
    compute: readCompute(settings),
    port: settings.port,
    command: settings.command.join(" "),
    healthcheck: {
      method: settings.healthcheck?.method ?? "GET",
      path: settings.healthcheck?.path ?? "",
      intervalSeconds: settings.healthcheck?.intervalSeconds ?? INTERVAL_SECONDS.default,
    },
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
  if (!sameJson(values.healthcheck, saved.healthcheck)) {
    writeHealthcheck(draft, values.healthcheck);
  }
}

function writeCommand(draft: EnvironmentSettings, command: string): void {
  const trimmed = command.trim();
  draft.command = trimmed === "" ? [] : trimmed.split(/\s+/);
}

function writeHealthcheck(draft: EnvironmentSettings, values: HealthcheckValues): void {
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
}
