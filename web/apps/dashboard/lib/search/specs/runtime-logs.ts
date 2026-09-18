import {
  runtimeLogsFilterFieldConfig,
  runtimeLogsFilterOutputSchema,
} from "@/lib/schemas/runtime-logs.filter.schema";
import type { SearchSpec } from "../spec";

export const runtimeLogsSearchSpec: SearchSpec = {
  subject: "runtime log lines",
  outputSchema: runtimeLogsFilterOutputSchema,
  config: runtimeLogsFilterFieldConfig,
  durationField: "since",
  fields: {
    severity: "how serious the log line is",
    message: "words written in the body of the log line",
    attributes: "a structured field attached to the line, written as path = value",
    region: "the edge region that produced the line, such as iad1 or fra1",
    appId: "the identifier of an application",
    deploymentId: "the identifier of a deployment",
    environmentId: "the identifier of an environment",
    instanceId: "the identifier of one running instance",
    since: "how far back the search reaches, as a lookback from now",
    startTime: "the start of a named range, as a millisecond timestamp",
    endTime: "the end of a named range, as a millisecond timestamp",
  },
  rules: [
    "Severity is ERROR, WARN, INFO or DEBUG. Read errors as ERROR and warnings as WARN, and include every level the query names.",
    "A word the user wants to find inside the text of a line goes in message with the contains operator.",
    "An attributes value must be written as path = value. Leave the field out when the query gives no path.",
  ],
  examples: [
    {
      query: "errors in the last hour",
      result: [
        { field: "severity", filters: [{ operator: "is", value: "ERROR" }] },
        { field: "since", filters: [{ operator: "is", value: "1h" }] },
      ],
    },
    {
      query: "warnings and errors from the past 3 days",
      result: [
        {
          field: "severity",
          filters: [
            { operator: "is", value: "ERROR" },
            { operator: "is", value: "WARN" },
          ],
        },
        { field: "since", filters: [{ operator: "is", value: "3d" }] },
      ],
      note: "both levels are named, so both appear",
    },
    {
      query: "log lines mentioning timeout",
      result: [{ field: "message", filters: [{ operator: "contains", value: "timeout" }] }],
    },
    {
      query: "debug lines from fra1",
      result: [
        { field: "severity", filters: [{ operator: "is", value: "DEBUG" }] },
        { field: "region", filters: [{ operator: "is", value: "fra1" }] },
      ],
    },
    {
      query: "everything in the last 30m",
      result: [{ field: "since", filters: [{ operator: "is", value: "30m" }] }],
    },
  ],
};
