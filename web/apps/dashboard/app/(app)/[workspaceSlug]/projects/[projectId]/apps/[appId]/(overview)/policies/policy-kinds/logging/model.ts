import { IconSquareBulletListOutline18 } from "@unkey/icons";
import { z } from "zod";
import { sharedFormFields } from "../shared";
import type { KindFields, PolicyKindModel } from "../types";

export const loggingFormSchema = z.object({
  ...sharedFormFields,
  type: z.literal("logging"),
  requestHeaders: z.boolean(),
  responseHeaders: z.boolean(),
  requestBody: z.boolean(),
  responseBody: z.boolean(),
  query: z.boolean(),
});

export type LoggingFormValues = z.infer<typeof loggingFormSchema>;
export type LoggingFieldName = Exclude<keyof KindFields<LoggingFormValues>, "type">;

type LoggingField = { name: LoggingFieldName; label: string; summary: string; hint?: string };

export const LOGGING_FIELDS: readonly LoggingField[] = [
  {
    name: "requestHeaders",
    label: "Request headers",
    summary: "request headers",
    hint: "Also stores the user agent and the client IP.",
  },
  { name: "responseHeaders", label: "Response headers", summary: "response headers" },
  { name: "requestBody", label: "Request body", summary: "request body" },
  { name: "responseBody", label: "Response body", summary: "response body" },
  { name: "query", label: "Query string", summary: "query" },
];

export const logging: PolicyKindModel<"logging", LoggingFormValues> = {
  label: "Logging",
  Icon: IconSquareBulletListOutline18,
  does: "Saves headers, query parameters or bodies of matching requests to the request log.",
  rejects: [],
  defaults: () => ({
    type: "logging",
    requestHeaders: true,
    responseHeaders: true,
    requestBody: true,
    responseBody: true,
    query: true,
  }),
  toWire: (v) => ({
    type: "logging",
    logging: {
      requestHeaders: v.requestHeaders,
      responseHeaders: v.responseHeaders,
      requestBody: v.requestBody,
      responseBody: v.responseBody,
      query: v.query,
    },
  }),
  // protojson omits false booleans.
  fromWire: (p) => ({
    type: "logging",
    requestHeaders: p.logging.requestHeaders ?? false,
    responseHeaders: p.logging.responseHeaders ?? false,
    requestBody: p.logging.requestBody ?? false,
    responseBody: p.logging.responseBody ?? false,
    query: p.logging.query ?? false,
  }),
  summary: (v) => {
    const on = LOGGING_FIELDS.filter(({ name }) => v[name]).map(({ summary }) => summary);
    return on.length > 0 ? on.join(", ") : "Default fields only";
  },
};
