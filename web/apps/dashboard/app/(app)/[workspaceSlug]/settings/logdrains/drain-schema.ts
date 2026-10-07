import type { Logdrain } from "@unkey/api/models/components";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import { z } from "zod";
import { headerNamePattern, isValidHttpHeaderValue } from "./header-fields";

export type DrainListItem = Logdrain;
export type DrainDetail = Logdrain;
export type DrainKind = "http" | "axiom";

const resourceIdsSchema = z.array(z.string().trim().min(1).max(256)).max(256);
const outcomesSchema = z.array(z.enum(KEY_VERIFICATION_OUTCOMES)).max(256);
const keySpaceIdsSchema = resourceIdsSchema;
const severitiesSchema = z.array(z.string().trim().min(1).max(256)).max(256);
const passedSchema = z.array(z.boolean()).max(2);
const statusClassesSchema = z.array(z.enum(["2xx", "3xx", "4xx", "5xx"])).max(4);

const headerRowSchema = z.object({
  name: z.string().max(256, "Header name must be 256 characters or less"),
  value: z
    .string()
    .max(8192, "Header value must be 8,192 characters or less")
    .refine(isValidHttpHeaderValue, "Enter a valid header value"),
  stored: z.boolean(),
});

type HeaderRow = z.infer<typeof headerRowSchema>;

function refineHeaderRows(rows: HeaderRow[], context: z.RefinementCtx) {
  const names = new Set<string>();
  const at = (index: number, field: "name" | "value") => ["headers", index, field];

  for (const [index, row] of rows.entries()) {
    const name = row.name.trim();
    if (name === "" && row.value === "" && !row.stored) {
      continue;
    }
    if (name === "") {
      context.addIssue({
        code: "custom",
        path: at(index, "name"),
        message: "Enter a header name",
      });
      continue;
    }
    if (!headerNamePattern.test(name)) {
      context.addIssue({
        code: "custom",
        path: at(index, "name"),
        message: "Enter a valid header name",
      });
    }
    if (row.value === "" && !row.stored) {
      context.addIssue({
        code: "custom",
        path: at(index, "value"),
        message: "Enter a header value",
      });
    }
    const normalized = name.toLowerCase();
    if (names.has(normalized)) {
      context.addIssue({
        code: "custom",
        path: at(index, "name"),
        message: "Header name is duplicated",
      });
    }
    names.add(normalized);
  }
}

const httpsUrlSchema = z
  .string()
  .url("Enter a valid URL")
  .superRefine((value, context) => {
    if (!URL.canParse(value)) {
      return;
    }
    const url = new URL(value);
    if (url.protocol !== "https:") {
      context.addIssue({ code: "custom", message: "URL must use HTTPS" });
    }
    if (url.username !== "" || url.password !== "") {
      context.addIssue({
        code: "custom",
        message: "URL must not contain credentials",
      });
    }
  });

const baseSchema = z.object({
  kind: z.enum(["http", "axiom"]),
  stream: z.enum([
    "audit_logs",
    "key_verifications",
    "gateway_requests",
    "runtime_logs",
    "ratelimits",
  ]),
  namespaceIds: resourceIdsSchema,
  passed: passedSchema,
  outcomes: outcomesSchema,
  keySpaceIds: keySpaceIdsSchema,
  statusClasses: statusClassesSchema,
  severities: severitiesSchema,
  runtimeProjectIds: resourceIdsSchema,
  runtimeAppIds: resourceIdsSchema,
  runtimeEnvironmentIds: resourceIdsSchema,
  runtimeSourceMode: z.enum(["all", "some"]),
  projectIds: resourceIdsSchema,
  appIds: resourceIdsSchema,
  environmentIds: resourceIdsSchema,
  name: z.string().trim().min(1, "Enter a name").max(128, "Name must be 128 characters or less"),
  url: z.string(),
  format: z.enum(["json", "ndjson", "hec"]),
  headers: z.array(headerRowSchema).max(32, "A maximum of 32 headers is supported"),
  dataset: z.string(),
  token: z.string(),
  eventTypes: z.array(z.string().trim().min(1).max(256)).max(256),
  sourceMode: z.enum(["all", "some"]),
  statusMode: z.enum(["all", "custom"]),
  eventTypesMode: z.enum(["all", "specific"]),
});

export type DrainFormValues = z.infer<typeof baseSchema>;

function refineDestination(
  values: DrainFormValues,
  context: z.RefinementCtx,
  { tokenRequired }: { tokenRequired: boolean },
) {
  switch (values.kind) {
    case "http": {
      const url = httpsUrlSchema.safeParse(values.url);
      if (!url.success) {
        context.addIssue({
          code: "custom",
          path: ["url"],
          message: url.error.issues[0]?.message ?? "Enter a valid HTTPS URL",
        });
      }
      refineHeaderRows(values.headers, context);
      break;
    }
    case "axiom":
      if (values.dataset.trim() === "") {
        context.addIssue({
          code: "custom",
          path: ["dataset"],
          message: "Enter a dataset",
        });
      }
      if (tokenRequired && values.token.trim() === "") {
        context.addIssue({
          code: "custom",
          path: ["token"],
          message: "Enter a token",
        });
      }
      break;
    default:
      throw new Error(`Unsupported log drain sink: ${values.kind satisfies never}`);
  }
}

function drainSchema({ tokenRequired }: { tokenRequired: boolean }) {
  return baseSchema.superRefine((values, context) => {
    refineDestination(values, context, { tokenRequired });
    if (
      values.stream === "audit_logs" &&
      values.eventTypesMode === "specific" &&
      values.eventTypes.length === 0
    ) {
      context.addIssue({
        code: "custom",
        path: ["eventTypes"],
        message: "Choose at least one event type",
      });
    }
    if (values.stream !== "gateway_requests" && values.stream !== "runtime_logs") {
      return;
    }
    const runtime = values.stream === "runtime_logs";
    const sources = submittedSources(values);
    const chosen =
      sources.projectIds.length + sources.appIds.length + sources.environmentIds.length;
    if ((runtime ? values.runtimeSourceMode : values.sourceMode) === "some" && chosen === 0) {
      context.addIssue({
        code: "custom",
        path: [runtime ? "runtimeEnvironmentIds" : "environmentIds"],
        message: "Choose at least one source",
      });
    }
    if (!runtime && values.statusMode === "custom" && values.statusClasses.length === 0) {
      context.addIssue({
        code: "custom",
        path: ["statusClasses"],
        message: "Choose at least one status class",
      });
    }
  });
}

export const createDrainSchema = drainSchema({ tokenRequired: true });

/** Editing keeps the stored token when the field is left blank. */
export const editDrainSchema = drainSchema({ tokenRequired: false });

export function headersChanged(submitted: HeaderRow[], current: HeaderRow[]): boolean {
  const headers = submitted.filter((header) => header.name.trim() !== "");
  return (
    headers.length !== current.length ||
    headers.some(
      (header, index) => header.name.trim() !== current[index]?.name || header.value !== "",
    )
  );
}

export function editDrainSchemaFor(current: DrainFormValues) {
  return editDrainSchema.superRefine((values, context) => {
    if (values.kind !== "http" || !headersChanged(values.headers, current.headers)) {
      return;
    }
    for (const [index, header] of values.headers.entries()) {
      if (header.name.trim() !== "" && header.value === "") {
        context.addIssue({
          code: "custom",
          path: ["headers", index, "value"],
          message: "Enter a value for every header you keep when changing headers",
        });
      }
    }
  });
}

export function submittedStatusClasses(values: DrainFormValues): DrainFormValues["statusClasses"] {
  switch (values.statusMode) {
    case "all":
      return [];
    case "custom":
      return values.statusClasses;
    default:
      throw new Error(`Unsupported status mode: ${values.statusMode satisfies never}`);
  }
}

export function submittedSources(values: DrainFormValues): {
  projectIds: string[];
  appIds: string[];
  environmentIds: string[];
} {
  if (values.stream === "runtime_logs") {
    return values.runtimeSourceMode === "all"
      ? { projectIds: [], appIds: [], environmentIds: [] }
      : {
          projectIds: values.runtimeProjectIds,
          appIds: values.runtimeAppIds,
          environmentIds: values.runtimeEnvironmentIds,
        };
  }
  if (values.sourceMode === "all") {
    return { projectIds: [], appIds: [], environmentIds: [] };
  }
  return {
    projectIds: values.projectIds,
    appIds: values.appIds,
    environmentIds: values.environmentIds,
  };
}

function statusModeFor(
  statusClasses: DrainFormValues["statusClasses"],
): DrainFormValues["statusMode"] {
  return statusClasses.length === 0 ? "all" : "custom";
}

export function submittedEventTypes(values: DrainFormValues): string[] {
  return values.eventTypesMode === "all" ? [] : values.eventTypes;
}

export function submittedStream(values: DrainFormValues) {
  switch (values.stream) {
    case "audit_logs":
      return { auditLogs: { eventTypes: submittedEventTypes(values) } };
    case "key_verifications":
      return { keyVerifications: { outcomes: values.outcomes, keySpaceIds: values.keySpaceIds } };
    case "ratelimits":
      return { ratelimits: { namespaceIds: values.namespaceIds, passed: values.passed } };
    case "gateway_requests":
      return {
        gatewayRequests: {
          statusClasses: submittedStatusClasses(values),
          ...submittedSources(values),
        },
      };
    case "runtime_logs":
      return { runtimeLogs: { severities: values.severities, ...submittedSources(values) } };
    default:
      throw new Error(`Unsupported log drain stream: ${values.stream satisfies never}`);
  }
}

export const emptyHeaderRow = { name: "", value: "", stored: false };

export const emptyDrainForm: DrainFormValues = {
  kind: "http",
  stream: "audit_logs",
  namespaceIds: [],
  passed: [],
  outcomes: [],
  keySpaceIds: [],
  statusClasses: [],
  severities: [],
  runtimeProjectIds: [],
  runtimeAppIds: [],
  runtimeEnvironmentIds: [],
  runtimeSourceMode: "all",
  projectIds: [],
  appIds: [],
  environmentIds: [],
  name: "",
  url: "",
  format: "json",
  headers: [{ ...emptyHeaderRow }],
  dataset: "",
  token: "",
  eventTypes: [],
  sourceMode: "all",
  statusMode: "all",
  eventTypesMode: "all",
};

export function drainToFormValues(drain: DrainDetail): DrainFormValues {
  const filters = drain.filters;
  const projectIds = filters.projectIds ?? [];
  const appIds = filters.appIds ?? [];
  const environmentIds = filters.environmentIds ?? [];
  const statusClasses = statusClassesSchema.parse(filters.statusClasses ?? []);
  const eventTypes = filters.eventTypes ?? [];
  const hasSources = projectIds.length + appIds.length + environmentIds.length > 0;
  return {
    ...emptyDrainForm,
    kind: drain.destination.http ? "http" : "axiom",
    name: drain.name,
    stream: drain.stream,
    namespaceIds: filters.namespaceIds ?? [],
    passed: filters.passed ?? [],
    outcomes: outcomesSchema.parse(filters.outcomes ?? []),
    keySpaceIds: filters.keySpaceIds ?? [],
    statusClasses,
    severities: filters.severities ?? [],
    runtimeProjectIds: drain.stream === "runtime_logs" ? projectIds : [],
    runtimeAppIds: drain.stream === "runtime_logs" ? appIds : [],
    runtimeEnvironmentIds: drain.stream === "runtime_logs" ? environmentIds : [],
    runtimeSourceMode: drain.stream === "runtime_logs" && hasSources ? "some" : "all",
    projectIds: drain.stream === "gateway_requests" ? projectIds : [],
    appIds: drain.stream === "gateway_requests" ? appIds : [],
    environmentIds: drain.stream === "gateway_requests" ? environmentIds : [],
    sourceMode: drain.stream === "gateway_requests" && hasSources ? "some" : "all",
    statusMode: statusModeFor(statusClasses),
    eventTypes,
    eventTypesMode: eventTypes.length > 0 ? "specific" : "all",
    url: drain.destination.http?.url ?? "",
    format: drain.destination.http?.format ?? "json",
    headers:
      drain.destination.http?.headers.map((name) => ({ name, value: "", stored: true })) ?? [],
    dataset: drain.destination.axiom?.dataset ?? "",
  };
}
