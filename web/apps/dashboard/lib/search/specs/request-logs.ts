import {
  requestLogsFilterFieldConfig,
  requestLogsFilterOutputSchema,
} from "@/lib/schemas/request-logs.filter.schema";
import type { SearchSpec } from "../spec";
import { METHOD_RULES, PATH_RULES, STATUS_RULES } from "./logs";

export const requestLogsSearchSpec: SearchSpec = {
  subject: "deployment request logs",
  outputSchema: requestLogsFilterOutputSchema,
  config: requestLogsFilterFieldConfig,
  durationField: "since",
  fields: {
    methods: "the HTTP method the request used",
    status: "how the request turned out",
    paths: "the part of the URL after the domain, such as /v1/keys/verify",
    host: "the domain the request was sent to",
    region: "the edge region that served the request, such as iad1 or fra1",
    requestId: "the identifier of one single request",
    appId: "the identifier of an application",
    deploymentId: "the identifier of a deployment",
    environmentId: "the identifier of an environment, such as production or preview",
    since: "how far back the search reaches, as a lookback from now",
    startTime: "the start of a named range, as a millisecond timestamp",
    endTime: "the end of a named range, as a millisecond timestamp",
  },
  rules: [...STATUS_RULES, ...METHOD_RULES, ...PATH_RULES],
  examples: [
    {
      query: "failed requests in the last 2h",
      result: [
        {
          field: "status",
          filters: [
            { operator: "is", value: 400 },
            { operator: "is", value: 500 },
          ],
        },
        { field: "since", filters: [{ operator: "is", value: "2h" }] },
      ],
    },
    {
      query: "POST requests to /v1/keys",
      result: [
        { field: "methods", filters: [{ operator: "is", value: "POST" }] },
        { field: "paths", filters: [{ operator: "startsWith", value: "v1/keys" }] },
      ],
    },
    {
      query: "requests served from iad1",
      result: [{ field: "region", filters: [{ operator: "is", value: "iad1" }] }],
    },
    {
      query: "server errors from the past 7 days",
      result: [
        { field: "status", filters: [{ operator: "is", value: 500 }] },
        { field: "since", filters: [{ operator: "is", value: "7d" }] },
      ],
    },
    { query: "show every request", result: [], note: "nothing is named, so nothing is filtered" },
  ],
};
