import type { Router } from "@/lib/trpc/routers";
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
      context.addIssue({ code: "custom", path: at(index, "name"), message: "Enter a header name" });
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
      context.addIssue({ code: "custom", message: "URL must not contain credentials" });
    }
  });

const baseSchema = z.object({
  kind: z.enum(["http", "axiom"]),
  name: z.string().trim().min(1, "Enter a name").max(128, "Name must be 128 characters or less"),
  url: z.string(),
  format: z.enum(["json", "ndjson"]),
  headers: z.array(headerRowSchema).max(32, "A maximum of 32 headers is supported"),
  dataset: z.string(),
  token: z.string(),
  startFrom: z.enum(["now", "beginning"]),
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
        context.addIssue({ code: "custom", path: ["dataset"], message: "Enter a dataset" });
      }
      if (tokenRequired && values.token.trim() === "") {
        context.addIssue({ code: "custom", path: ["token"], message: "Enter a token" });
      }
      break;
    default:
      throw new Error(`Unsupported log drain sink: ${values.kind satisfies never}`);
  }
}

export const createDrainSchema = baseSchema.superRefine((values, context) =>
  refineDestination(values, context, { tokenRequired: true }),
);

/** Editing keeps the stored token when the field is left blank. */
export const editDrainSchema = baseSchema.superRefine((values, context) =>
  refineDestination(values, context, { tokenRequired: false }),
);

export const emptyHeaderRow = { name: "", value: "", stored: false };

export const emptyDrainForm: DrainFormValues = {
  kind: "http",
  name: "",
  url: "",
  format: "json",
  headers: [{ ...emptyHeaderRow }],
  dataset: "",
  token: "",
  startFrom: "now",
};

export function drainToFormValues(drain: DrainDetail): DrainFormValues {
  return {
    ...emptyDrainForm,
    kind: drain.kind,
    name: drain.name,
    url: drain.kind === "http" ? drain.config.url : "",
    format: drain.kind === "http" ? drain.config.format : "json",
    headers:
      drain.kind === "http"
        ? drain.config.headers.map((name) => ({ name, value: "", stored: true }))
        : [],
    dataset: drain.kind === "axiom" ? drain.config.dataset : "",
  };
}
