import {
  auditFilterOutputSchema,
  auditLogsFilterFieldConfig,
} from "@/app/(app)/[workspaceSlug]/audit/filters.schema";
import { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import type { SearchSpec } from "../spec";

export const auditSearchSpec: SearchSpec = {
  subject: "audit log entries",
  outputSchema: auditFilterOutputSchema,
  config: auditLogsFilterFieldConfig,
  durationField: "since",
  fields: {
    events: "the kind of change that was recorded",
    users: "the person who made the change",
    rootKeys: "the root key that made the change, when it came from the API",
    bucket: "the audit log bucket the entry was written to",
    since: "how far back the search reaches, as a lookback from now",
    startTime: "the start of a named range, as a millisecond timestamp",
    endTime: "the end of a named range, as a millisecond timestamp",
  },
  rules: [
    `Every event is named resource.action and must be one of: ${unkeyAuditLogEvents.options.join(", ")}.`,
    "When the query names a resource but no action, list every event for that resource.",
    "When the query names an action but no resource, list that action across every resource it exists on.",
  ],
  examples: [
    {
      query: "show key.create events",
      result: [{ field: "events", filters: [{ operator: "is", value: "key.create" }] }],
    },
    {
      query: "show all role deletions",
      result: [{ field: "events", filters: [{ operator: "is", value: "role.delete" }] }],
    },
    {
      query: "anything that happened to webhooks in the past 3 days",
      result: [
        {
          field: "events",
          filters: [
            { operator: "is", value: "webhook.create" },
            { operator: "is", value: "webhook.update" },
            { operator: "is", value: "webhook.delete" },
          ],
        },
        { field: "since", filters: [{ operator: "is", value: "3d" }] },
      ],
      note: "a resource with no action named covers every event for it",
    },
    {
      query: "who created a key in the last 24h",
      result: [
        { field: "events", filters: [{ operator: "is", value: "key.create" }] },
        { field: "since", filters: [{ operator: "is", value: "24h" }] },
      ],
    },
    {
      query: "find events for user user_123",
      result: [{ field: "users", filters: [{ operator: "is", value: "user_123" }] }],
    },
    { query: "show the audit log", result: [], note: "nothing is named, so nothing is filtered" },
  ],
};
