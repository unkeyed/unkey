export type ExpectedFilter = { field: string; operator: string; value: string | number };

export type GoldenQuery = {
  query: string;
  expected: ExpectedFilter[];
};

export const GOLDEN_QUERIES: GoldenQuery[] = [
  {
    query: "find all POST requests",
    expected: [{ field: "methods", operator: "is", value: "POST" }],
  },
  {
    query: "show requests with status 404",
    expected: [{ field: "status", operator: "is", value: 400 }],
  },
  {
    query: "get, post and delete requests",
    expected: [
      { field: "methods", operator: "is", value: "GET" },
      { field: "methods", operator: "is", value: "POST" },
      { field: "methods", operator: "is", value: "DELETE" },
    ],
  },
  {
    query: "show me timeouts and server errors",
    expected: [{ field: "status", operator: "is", value: 500 }],
  },
  {
    query: "client errors and failed requests",
    expected: [{ field: "status", operator: "is", value: 400 }],
  },
  {
    query: "show successful requests",
    expected: [{ field: "status", operator: "is", value: 200 }],
  },
  {
    query: "localhost requests",
    expected: [{ field: "host", operator: "is", value: "localhost" }],
  },
  {
    query: "show requests from api.staging.company.com",
    expected: [{ field: "host", operator: "is", value: "api.staging.company.com" }],
  },
  {
    query: "find requests to /api/v1",
    expected: [{ field: "paths", operator: "startsWith", value: "api/v1" }],
  },
  {
    query: "requests to paths ending in /verify",
    expected: [{ field: "paths", operator: "endsWith", value: "verify" }],
  },
  {
    query: "errors from the last 30m",
    expected: [
      { field: "status", operator: "is", value: 400 },
      { field: "status", operator: "is", value: 500 },
      { field: "since", operator: "is", value: "30m" },
    ],
  },
  {
    query: "show requests from last week",
    expected: [{ field: "since", operator: "is", value: "7d" }],
  },
  {
    query: "read operations to /api in the last 24h",
    expected: [
      { field: "methods", operator: "is", value: "GET" },
      { field: "paths", operator: "startsWith", value: "api" },
      { field: "since", operator: "is", value: "24h" },
    ],
  },
  {
    query: "failed requests to /v1/keys from api.prod.com in last 2h",
    expected: [
      { field: "status", operator: "is", value: 400 },
      { field: "status", operator: "is", value: 500 },
      { field: "paths", operator: "startsWith", value: "v1/keys" },
      { field: "host", operator: "is", value: "api.prod.com" },
      { field: "since", operator: "is", value: "2h" },
    ],
  },
  {
    query: "localhost GET and POST /api errors since 2h ago",
    expected: [
      { field: "host", operator: "is", value: "localhost" },
      { field: "methods", operator: "is", value: "GET" },
      { field: "methods", operator: "is", value: "POST" },
      { field: "paths", operator: "startsWith", value: "api" },
      { field: "status", operator: "is", value: 400 },
      { field: "status", operator: "is", value: 500 },
      { field: "since", operator: "is", value: "2h" },
    ],
  },
  {
    query: "write operations that crashed the server yesterday",
    expected: [
      { field: "methods", operator: "is", value: "POST" },
      { field: "status", operator: "is", value: 500 },
      { field: "startTime", operator: "is", value: 1769078400000 },
      { field: "endTime", operator: "is", value: 1769164800000 },
    ],
  },
  {
    query: "anything that went wrong in the past 3 days",
    expected: [
      { field: "status", operator: "is", value: 400 },
      { field: "status", operator: "is", value: 500 },
      { field: "since", operator: "is", value: "3d" },
    ],
  },
  {
    query: "requests where the path contains checkout",
    expected: [{ field: "paths", operator: "contains", value: "checkout" }],
  },
  {
    query: "delete calls that were rejected",
    expected: [
      { field: "methods", operator: "is", value: "DELETE" },
      { field: "status", operator: "is", value: 400 },
    ],
  },
  {
    query: "everything from 127.0.0.1 in the last hour",
    expected: [
      { field: "host", operator: "is", value: "127.0.0.1" },
      { field: "since", operator: "is", value: "1h" },
    ],
  },
];
