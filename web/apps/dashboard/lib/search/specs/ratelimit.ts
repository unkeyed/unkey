import {
  filterOutputSchema,
  ratelimitFilterFieldConfig,
} from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/logs/filters.schema";
import type { SearchSpec } from "../spec";

export const ratelimitSearchSpec: SearchSpec = {
  subject: "ratelimit decisions",
  outputSchema: filterOutputSchema,
  config: ratelimitFilterFieldConfig,
  durationField: "since",
  fields: {
    status: "whether the request was let through",
    identifiers:
      "the thing being rate limited, such as a user id, an API key, an email, or an IP address",
    since: "how far back the search reaches, as a lookback from now",
    startTime: "the start of a named range, as a millisecond timestamp",
    endTime: "the end of a named range, as a millisecond timestamp",
  },
  rules: [
    "blocked means the limit stopped the request, so read throttled and rate limited as blocked.",
    "passed means the request stayed inside the limit.",
    "An identifier given whole is matched exactly. A fragment is matched by contains.",
  ],
  examples: [
    {
      query: "show blocked requests",
      result: [{ field: "status", filters: [{ operator: "is", value: "blocked" }] }],
    },
    {
      query: "who got rate limited in the past 30m",
      result: [
        { field: "status", filters: [{ operator: "is", value: "blocked" }] },
        { field: "since", filters: [{ operator: "is", value: "30m" }] },
      ],
    },
    {
      query: "everything for user_abc123def",
      result: [{ field: "identifiers", filters: [{ operator: "is", value: "user_abc123def" }] }],
    },
    {
      query: "identifiers containing acme",
      result: [{ field: "identifiers", filters: [{ operator: "contains", value: "acme" }] }],
    },
    {
      query: "requests that were let through today",
      result: [
        { field: "status", filters: [{ operator: "is", value: "passed" }] },
        { field: "since", filters: [{ operator: "is", value: "24h" }] },
      ],
    },
    { query: "show all requests", result: [], note: "nothing is named, so nothing is filtered" },
  ],
};
