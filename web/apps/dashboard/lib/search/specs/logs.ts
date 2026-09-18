import { METHODS } from "@/app/(app)/[workspaceSlug]/logs/constants";
import { filterOutputSchema, logsFilterFieldConfig } from "@/lib/schemas/logs.filter.schema";
import type { SearchSpec } from "../spec";

export const STATUS_RULES = [
  "Status is written as one of three numbers only: 200 for success, 400 for anything the caller got wrong, 500 for anything the server got wrong.",
  'A term that could mean either kind of failure, such as "failed" or "broken", means both 400 and 500.',
  "Map a literal code to its group, so 404 becomes 400 and 503 becomes 500.",
];

export const METHOD_RULES = [
  `Methods are one of ${METHODS.join(", ")}.`,
  'Read "read" and "fetch" as GET, and "write", "send" and "create" as POST.',
];

export const PATH_RULES = [
  "Paths are written without a leading slash, so /v1/keys becomes v1/keys.",
  "A path named with no other qualifier means everything underneath it, so use startsWith.",
  "When a path holds a parameter such as {id}, match the part before it.",
];

export const logsSearchSpec: SearchSpec = {
  subject: "HTTP request logs",
  outputSchema: filterOutputSchema,
  config: logsFilterFieldConfig,
  durationField: "since",
  fields: {
    methods: "the HTTP method the request used",
    status: "how the request turned out",
    paths: "the part of the URL after the domain, such as /v1/keys/verify",
    host: "the domain or machine the request was sent to, such as api.unkey.dev or localhost",
    requestId: "the identifier of one single request",
    appId: "the identifier of an application",
    deploymentId: "the identifier of a deployment",
    environmentId: "the identifier of an environment, such as production or preview",
    since: "how far back the search reaches, as a lookback from now",
    startTime: "the start of a named range, as a millisecond timestamp",
    endTime: "the end of a named range, as a millisecond timestamp",
  },
  rules: [...STATUS_RULES, ...METHOD_RULES, ...PATH_RULES],
  priorities: [
    'A named method beats an implied one, so "GET" wins over "read-like".',
    'When a word could mean success or failure, prefer failure for terms like "failed" or "issues".',
  ],
  examples: [
    {
      query: "find all POST requests",
      result: [{ field: "methods", filters: [{ operator: "is", value: "POST" }] }],
    },
    {
      query: "show requests with status 404",
      result: [{ field: "status", filters: [{ operator: "is", value: 400 }] }],
      note: "404 belongs to the 400 group",
    },
    {
      query: "show me timeouts and server errors",
      result: [{ field: "status", filters: [{ operator: "is", value: 500 }] }],
    },
    {
      query: "client errors and failed requests",
      result: [{ field: "status", filters: [{ operator: "is", value: 400 }] }],
    },
    {
      query: "localhost and 127.0.0.1 requests",
      result: [
        {
          field: "host",
          filters: [
            { operator: "is", value: "localhost" },
            { operator: "is", value: "127.0.0.1" },
          ],
        },
      ],
    },
    {
      query: "find /api/v2/users/{userId}/profile and /api/v2/users/search requests",
      result: [{ field: "paths", filters: [{ operator: "startsWith", value: "api/v2/users" }] }],
      note: "both share a base path, so one prefix covers them",
    },
    {
      query: "requests to paths ending in /verify",
      result: [{ field: "paths", filters: [{ operator: "endsWith", value: "verify" }] }],
    },
    {
      query: "errors from last 30m and last 24h",
      result: [
        {
          field: "status",
          filters: [
            { operator: "is", value: 400 },
            { operator: "is", value: 500 },
          ],
        },
        { field: "since", filters: [{ operator: "is", value: "24h" }] },
      ],
      note: "two windows are named, so the longest wins",
    },
    {
      query: "show requests from last week",
      result: [{ field: "since", filters: [{ operator: "is", value: "7d" }] }],
    },
    {
      query: "read and write operations to /api",
      result: [
        {
          field: "methods",
          filters: [
            { operator: "is", value: "GET" },
            { operator: "is", value: "POST" },
          ],
        },
        { field: "paths", filters: [{ operator: "startsWith", value: "api" }] },
      ],
    },
    {
      query: "failed requests to /v1/users or /v2/users from api.prod.com in last 2h",
      result: [
        {
          field: "status",
          filters: [
            { operator: "is", value: 400 },
            { operator: "is", value: 500 },
          ],
        },
        {
          field: "paths",
          filters: [
            { operator: "startsWith", value: "v1/users" },
            { operator: "startsWith", value: "v2/users" },
          ],
        },
        { field: "host", filters: [{ operator: "is", value: "api.prod.com" }] },
        { field: "since", filters: [{ operator: "is", value: "2h" }] },
      ],
    },
    {
      query: "errors between 9am and 11am today",
      result: [
        {
          field: "status",
          filters: [
            { operator: "is", value: 400 },
            { operator: "is", value: 500 },
          ],
        },
        { field: "startTime", filters: [{ operator: "is", value: 1769158800000 }] },
        { field: "endTime", filters: [{ operator: "is", value: 1769166000000 }] },
      ],
      note: "a named range uses startTime and endTime, never since",
    },
    {
      query: "show me everything",
      result: [],
      note: "nothing is named, so nothing is filtered",
    },
  ],
};
