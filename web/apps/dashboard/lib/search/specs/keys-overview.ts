import {
  filterOutputSchema,
  keysOverviewFilterFieldConfig,
} from "@/app/(app)/[workspaceSlug]/apis/[apiId]/_overview/filters.schema";
import type { SearchSpec } from "../spec";
import { KEY_FIELDS } from "./keys-list";
import { NAME_RULES } from "./roles";

export const keysOverviewSearchSpec: SearchSpec = {
  subject: "key verifications",
  outputSchema: filterOutputSchema,
  config: keysOverviewFilterFieldConfig,
  durationField: "since",
  fields: {
    ...KEY_FIELDS,
    outcomes: "how the key verification turned out",
    since: "how far back the search reaches, as a lookback from now",
    startTime: "the start of a named range, as a millisecond timestamp",
    endTime: "the end of a named range, as a millisecond timestamp",
  },
  rules: [
    ...NAME_RULES,
    "RATE_LIMITED means too many requests in a short window. USAGE_EXCEEDED means the key spent the total number of uses it was ever given.",
    "INSUFFICIENT_PERMISSIONS means the key was real but not allowed. FORBIDDEN means it was refused outright. DISABLED means it was switched off.",
  ],
  examples: [
    {
      query: "rate limited verifications in the last 24h",
      result: [
        { field: "outcomes", filters: [{ operator: "is", value: "RATE_LIMITED" }] },
        { field: "since", filters: [{ operator: "is", value: "24h" }] },
      ],
    },
    {
      query: "keys that ran out of their allowance",
      result: [{ field: "outcomes", filters: [{ operator: "is", value: "USAGE_EXCEEDED" }] }],
    },
    {
      query: "successful verifications",
      result: [{ field: "outcomes", filters: [{ operator: "is", value: "VALID" }] }],
    },
    {
      query: "verifications that were blocked",
      result: [
        {
          field: "outcomes",
          filters: [
            { operator: "is", value: "FORBIDDEN" },
            { operator: "is", value: "DISABLED" },
            { operator: "is", value: "RATE_LIMITED" },
            { operator: "is", value: "INSUFFICIENT_PERMISSIONS" },
          ],
        },
      ],
      note: "blocked names no single outcome, so include every way a key is refused",
    },
    {
      query: "expired keys tagged internal in the past 7 days",
      result: [
        { field: "outcomes", filters: [{ operator: "is", value: "EXPIRED" }] },
        { field: "tags", filters: [{ operator: "is", value: "internal" }] },
        { field: "since", filters: [{ operator: "is", value: "7d" }] },
      ],
    },
    { query: "show everything", result: [], note: "nothing is named, so nothing is filtered" },
  ],
};
