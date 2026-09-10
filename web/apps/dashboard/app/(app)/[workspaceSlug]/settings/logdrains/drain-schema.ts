import type { Router } from "@/lib/trpc/routers";
import {
  keySpaceIdsSchema,
  outcomesSchema,
  resourceIdsSchema,
  severitiesSchema,
  statusClassesSchema,
} from "@/lib/trpc/routers/logdrain/validation";
import type { inferRouterOutputs } from "@trpc/server";
import { z } from "zod";
import { headerNamePattern, isValidHttpHeaderValue } from "./header-fields";

type Outputs = inferRouterOutputs<Router>;
export type DrainListItem = Outputs["logdrain"]["list"][number];
export type DrainDetail = Outputs["logdrain"]["get"];
export type DrainKind = DrainListItem["kind"];

/**
 * A stored header keeps its encrypted value on the server, so an empty value on one of those
 * rows means "leave it alone" rather than "clear it". New rows have no stored value to fall
 * back on and must carry one.
 */
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
  stream: z.enum(["audit_logs", "key_verifications", "gateway_requests", "runtime_logs"]),
  outcomes: outcomesSchema,
  keySpaceIds: keySpaceIdsSchema,
  statusClasses: statusClassesSchema,
  severities: severitiesSchema,
  runtimeProjectIds: resourceIdsSchema,
  runtimeAppIds: resourceIdsSchema,
  runtimeEnvironmentIds: resourceIdsSchema,
  projectIds: resourceIdsSchema,
  appIds: resourceIdsSchema,
  environmentIds: resourceIdsSchema,
  name: z.string().trim().min(1, "Enter a name").max(128, "Name must be 128 characters or less"),
  url: z.string(),
  format: z.enum(["json", "ndjson"]),
  headers: z.array(headerRowSchema).max(32, "A maximum of 32 headers is supported"),
  dataset: z.string(),
  token: z.string(),
  eventTypes: z.array(z.string().trim().min(1).max(256)).max(256),
  sourceMode: z.enum(["all", "some"]),
  statusMode: z.enum(["all", "errors", "custom"]),
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
    if (values.stream !== "gateway_requests") {
      return;
    }
    const sources = submittedSources(values);
    const chosen =
      sources.projectIds.length + sources.appIds.length + sources.environmentIds.length;
    if (values.sourceMode === "some" && chosen === 0) {
      context.addIssue({
        code: "custom",
        path: ["environmentIds"],
        message: "Choose at least one source",
      });
    }
    if (values.statusMode === "custom" && values.statusClasses.length === 0) {
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

export const ERROR_STATUS_CLASSES = [4, 5];

export function submittedStatusClasses(values: DrainFormValues): number[] {
  switch (values.statusMode) {
    case "all":
      return [];
    case "errors":
      return [...ERROR_STATUS_CLASSES];
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
  if (values.sourceMode === "all") {
    return { projectIds: [], appIds: [], environmentIds: [] };
  }
  return {
    projectIds: values.projectIds,
    appIds: values.appIds,
    environmentIds: values.environmentIds,
  };
}

function statusModeFor(statusClasses: number[]): DrainFormValues["statusMode"] {
  if (statusClasses.length === 0) {
    return "all";
  }
  const sorted = [...statusClasses].sort();
  return sorted.length === ERROR_STATUS_CLASSES.length &&
    sorted.every((statusClass, index) => statusClass === ERROR_STATUS_CLASSES[index])
    ? "errors"
    : "custom";
}

export function submittedEventTypes(values: DrainFormValues): string[] {
  return values.eventTypesMode === "all" ? [] : values.eventTypes;
}

export const emptyHeaderRow = { name: "", value: "", stored: false };

export const emptyDrainForm: DrainFormValues = {
  kind: "http",
  stream: "audit_logs",
  outcomes: [],
  keySpaceIds: [],
  statusClasses: [],
  severities: [],
  runtimeProjectIds: [],
  runtimeAppIds: [],
  runtimeEnvironmentIds: [],
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
  return {
    ...emptyDrainForm,
    kind: drain.kind,
    name: drain.name,
    stream: drain.stream,
    outcomes: outcomesSchema.parse(drain.outcomes),
    keySpaceIds: drain.keySpaceIds,
    statusClasses: statusClassesSchema.parse(drain.statusClasses),
    severities: drain.severities,
    runtimeProjectIds: drain.stream === "runtime_logs" ? drain.projectIds : [],
    runtimeAppIds: drain.stream === "runtime_logs" ? drain.appIds : [],
    runtimeEnvironmentIds: drain.stream === "runtime_logs" ? drain.environmentIds : [],
    projectIds: drain.stream === "gateway_requests" ? drain.projectIds : [],
    appIds: drain.stream === "gateway_requests" ? drain.appIds : [],
    environmentIds: drain.stream === "gateway_requests" ? drain.environmentIds : [],
    sourceMode:
      drain.stream === "gateway_requests" &&
      drain.projectIds.length + drain.appIds.length + drain.environmentIds.length > 0
        ? "some"
        : "all",
    statusMode: statusModeFor(statusClassesSchema.parse(drain.statusClasses)),
    eventTypes: drain.eventTypes,
    eventTypesMode: drain.eventTypes.length > 0 ? "specific" : "all",
    url: drain.kind === "http" ? drain.config.url : "",
    format: drain.kind === "http" ? drain.config.format : "json",
    headers:
      drain.kind === "http"
        ? drain.config.headers.map((name) => ({
            name,
            value: "",
            stored: true,
          }))
        : [],
    dataset: drain.kind === "axiom" ? drain.config.dataset : "",
  };
}
