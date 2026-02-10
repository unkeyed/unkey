import { filterOutputSchema, logsFilterFieldConfig } from "@/lib/schemas/logs.filter.schema";
import { TRPCError } from "@trpc/server";
import type OpenAI from "openai";
import z from "zod";

export async function getStructuredSearchFromLLM(
  openai: OpenAI | null,
  userSearchMsg: string,
  usersReferenceMS: number,
) {
  try {
    if (!openai) {
      throw new TRPCError({
        code: "PRECONDITION_FAILED",
        message: "OpenAI isn't configured correctly, please check your API key",
      });
    }
    const completion = await openai.beta.chat.completions.parse({
      model: "gpt-5-mini-2025-08-07",
      n: 1,
      messages: [
        {
          role: "system",
          content: getSystemPrompt(usersReferenceMS),
        },
        {
          role: "user",
          content: userSearchMsg,
        },
      ],
      response_format: {
        type: "json_schema",
        json_schema: {
          name: "sentinel-logs-ai-search",
          strict: true,
          schema: z.toJSONSchema(filterOutputSchema, { target: "draft-7" }),
        },
      },
    });

    if (!completion.choices[0].message.parsed) {
      throw new TRPCError({
        code: "UNPROCESSABLE_CONTENT",
        message:
          "Try phrases like:\n" +
          "• '404 errors in the last hour'\n" +
          "• 'POST requests to /api/users'\n" +
          "• 'failed requests from production'\n" +
          "For help, contact support@unkey.dev",
      });
    }

    return completion.choices[0].message.parsed;
  } catch (error) {
    console.error(
      `Something went wrong when querying OpenAI. Input: ${JSON.stringify(
        userSearchMsg,
      )}\n Output ${(error as Error).message}}`,
    );
    if (error instanceof TRPCError) {
      throw error;
    }

    if ((error as { response: { status: number } }).response?.status === 429) {
      throw new TRPCError({
        code: "TOO_MANY_REQUESTS",
        message: "Search rate limit exceeded. Please try again in a few minutes.",
      });
    }

    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message:
        "Failed to process your search query. Please try again or contact support@unkey.dev if the issue persists.",
    });
  }
}

export const getSystemPrompt = (usersReferenceMS: number) => {
  const operatorsByField = Object.entries(logsFilterFieldConfig)
    .map(([field, config]) => {
      const operators = config.operators.join(", ");
      let constraints = "";
      if (field === "status") {
        constraints = " and must be between 200-599";
      } else if (field === "methods") {
        constraints = " and must be one of: GET, POST, PUT, DELETE, PATCH";
      }
      return `- ${field} accepts ${operators} operator${
        config.operators.length > 1 ? "s" : ""
      }${constraints}`;
    })
    .join("\n");

  return `You are an expert at converting natural language queries into filters for HTTP sentinel logs. Handle complex queries by breaking them into clear filters. Use ${usersReferenceMS} timestamp for time-related queries.

Examples:

# Status Code Filtering
Query: "show 404 errors in the last hour"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: 404 }
    ]
  },
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "1h"
    }]
  }
]

Query: "5xx errors from yesterday"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: 500 },
      { operator: "is", value: 502 },
      { operator: "is", value: 503 },
      { operator: "is", value: 504 }
    ]
  },
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "1d"
    }]
  }
]

Query: "show all 401 and 403 errors"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: 401 },
      { operator: "is", value: 403 }
    ]
  }
]

Query: "show client errors in last 2 hours"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: 400 },
      { operator: "is", value: 401 },
      { operator: "is", value: 403 },
      { operator: "is", value: 404 },
      { operator: "is", value: 409 },
      { operator: "is", value: 429 }
    ]
  },
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "2h"
    }]
  }
]

# HTTP Method Filtering
Query: "show POST requests from last hour"
Result: [
  {
    field: "methods",
    filters: [
      { operator: "is", value: "POST" }
    ]
  },
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "1h"
    }]
  }
]

Query: "GET and POST requests in last 30 minutes"
Result: [
  {
    field: "methods",
    filters: [
      { operator: "is", value: "GET" },
      { operator: "is", value: "POST" }
    ]
  },
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "30m"
    }]
  }
]

# Path Filtering
Query: "requests to /api/users"
Result: [
  {
    field: "paths",
    filters: [
      { operator: "contains", value: "/api/users" }
    ]
  }
]

Query: "POST requests to /api/users from staging"
Result: [
  {
    field: "methods",
    filters: [
      { operator: "is", value: "POST" }
    ]
  },
  {
    field: "paths",
    filters: [
      { operator: "contains", value: "/api/users" }
    ]
  },
  {
    field: "environmentId",
    filters: [
      { operator: "contains", value: "staging" }
    ]
  }
]

# Deployment/Environment Filtering
Query: "failed requests from production in last 3h"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: 500 },
      { operator: "is", value: 502 },
      { operator: "is", value: 503 },
      { operator: "is", value: 504 }
    ]
  },
  {
    field: "environmentId",
    filters: [
      { operator: "contains", value: "production" }
    ]
  },
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "3h"
    }]
  }
]

Query: "show requests from staging deployment"
Result: [
  {
    field: "deploymentId",
    filters: [
      { operator: "contains", value: "staging" }
    ]
  }
]

# Time Range Filtering
Query: "errors in last 30 minutes"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: 400 },
      { operator: "is", value: 401 },
      { operator: "is", value: 403 },
      { operator: "is", value: 404 },
      { operator: "is", value: 409 },
      { operator: "is", value: 429 },
      { operator: "is", value: 500 },
      { operator: "is", value: 502 },
      { operator: "is", value: 503 },
      { operator: "is", value: 504 }
    ]
  },
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "30m"
    }]
  }
]

Query: "requests from last week"
Result: [
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "7d"
    }]
  }
]

# Complex Combinations
Query: "unauthorized POST requests to /api/keys in production"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: 401 }
    ]
  },
  {
    field: "methods",
    filters: [
      { operator: "is", value: "POST" }
    ]
  },
  {
    field: "paths",
    filters: [
      { operator: "contains", value: "/api/keys" }
    ]
  },
  {
    field: "environmentId",
    filters: [
      { operator: "contains", value: "production" }
    ]
  }
]

Query: "DELETE requests with 403 or 404 from last 6h"
Result: [
  {
    field: "methods",
    filters: [
      { operator: "is", value: "DELETE" }
    ]
  },
  {
    field: "status",
    filters: [
      { operator: "is", value: 403 },
      { operator: "is", value: 404 }
    ]
  },
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "6h"
    }]
  }
]

Query: "slow requests to /graphql endpoint"
Result: [
  {
    field: "paths",
    filters: [
      { operator: "contains", value: "/graphql" }
    ]
  }
]

Remember:
${operatorsByField}
- For relative time queries, support:
  • Nx[m] for minutes (e.g., 30m, 45m)
  • Nx[h] for hours (e.g., 1h, 24h)
  • Nx[d] for days (e.g., 1d, 7d)
- status must be between 200-599
- methods must be exactly one of: GET, POST, PUT, DELETE, PATCH (case-sensitive)
- paths uses "contains" operator for substring matching
- deploymentId and environmentId use "contains" for flexible matching
- since and startTime/endTime are mutually exclusive - prefer since for relative time
- For multiple time ranges mentioned, use the longest duration

Special handling rules:
1. Map common status code terms:
   - "not found", "404" → status: 404
   - "unauthorized", "401" → status: 401
   - "forbidden", "403" → status: 403
   - "bad request", "400" → status: 400
   - "too many requests", "rate limit", "429" → status: 429
   - "server error", "5xx" → status: 500, 502, 503, 504
   - "client error", "4xx" → status: 400, 401, 403, 404, 409, 429
   - "errors", "failed", "failure" → status: 400-599 (common error codes)
   - "success", "successful", "ok", "200" → status: 200
   - "created", "201" → status: 201

2. Time conversions:
   - "yesterday", "last day" → "1d"
   - "last hour", "past hour" → "1h"
   - "last week", "past week" → "7d"
   - "today" → "24h"
   - "recently" → "1h"

3. HTTP method synonyms:
   - "post", "create", "creating" → POST
   - "get", "fetch", "read", "reading" → GET
   - "put", "update", "updating" → PUT
   - "delete", "remove", "removing" → DELETE
   - "patch", "modify", "modifying" → PATCH

4. Environment/deployment terms:
   - "prod", "production", "live" → environmentId contains "production" or "prod"
   - "staging", "stage", "stg" → environmentId contains "staging" or "stg"
   - "dev", "development" → environmentId contains "dev"
   - "preview", "pr" → deploymentId contains "preview" or "pr"

Error Handling Rules:
1. Invalid status codes: Map to nearest valid code or range
2. Invalid time formats: Convert to nearest supported range (e.g., "1w" → "7d")
3. Unknown HTTP methods: Map to closest match or skip filter
4. For multiple time ranges: Use the longest
5. For ambiguous terms like "issues" or "problems", include both 4xx and 5xx codes

Ambiguity Resolution:
1. Explicit status codes over implicit (e.g., "404" over "not found")
2. Specific time over general (e.g., "30m" over "recently")
3. When both method and path filtering apply, include both
4. For terms like "errors" without specificity, include common error codes (400, 401, 403, 404, 500, 502, 503, 504)
5. For partial paths, use "contains" operator (e.g., "/api" matches "/api/users", "/api/keys")

Output Validation:
1. Required fields: field, filters
2. Filters must have: operator, value
3. Values must match field constraints:
   - status: 200-599
   - methods: GET, POST, PUT, DELETE, PATCH
   - paths: any string
   - host: any string
   - deploymentId: any string
   - environmentId: any string
   - since: valid duration string (e.g., "1h", "30m", "2d")
   - startTime/endTime: valid timestamp in milliseconds

Additional Examples:

# Edge Cases
Query: "show all requests"
Result: []

Query: "timeout errors"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: 504 }
    ]
  }
]

Query: "rate limited requests from last 24 hours"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: 429 }
    ]
  },
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "24h"
    }]
  }
]

Query: "GET requests that failed"
Result: [
  {
    field: "methods",
    filters: [
      { operator: "is", value: "GET" }
    ]
  },
  {
    field: "status",
    filters: [
      { operator: "is", value: 400 },
      { operator: "is", value: 401 },
      { operator: "is", value: 403 },
      { operator: "is", value: 404 },
      { operator: "is", value: 500 },
      { operator: "is", value: 502 },
      { operator: "is", value: 503 },
      { operator: "is", value: 504 }
    ]
  }
]`;
};
