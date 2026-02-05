import { METHODS } from "@/app/(app)/[workspaceSlug]/logs/constants";
import { filterOutputSchema, logsFilterFieldConfig } from "@/lib/schemas/logs.filter.schema";
import { TRPCError } from "@trpc/server";
import type OpenAI from "openai";
import { zodResponseFormat } from "openai/helpers/zod.mjs";

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
      // Don't change the model only a few models allow structured outputs
      model: "gpt-4o-mini",
      temperature: 0.1, // Range 0-2, lower = more focused/deterministic
      n: 1, // Number of completions to generate
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
      response_format: zodResponseFormat(filterOutputSchema, "searchQuery"),
    });

    if (!completion.choices[0].message.parsed) {
      throw new TRPCError({
        code: "UNPROCESSABLE_CONTENT",
        message:
          "Try using phrases like:\n" +
          "• 'find all POST requests'\n" +
          "• 'show requests with status 404'\n" +
          "• 'find requests to api/v1'\n" +
          "• 'show requests from test.example.com'\n" +
          "• 'find all GET and POST requests'\n" +
          "For additional help, contact support@unkey.com",
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
        "Failed to process your search query. Please try again or contact support@unkey.com if the issue persists.",
    });
  }
}
export const getSystemPrompt = (usersReferenceMS: number) => {
  const operatorsByField = Object.entries(logsFilterFieldConfig)
    .map(([field, config]) => {
      const operators = config.operators.join(", ");
      let constraints = "";
      if (field === "methods") {
        constraints = ` and must be one of: ${METHODS.join(", ")}`;
      } else if (field === "status") {
        constraints = " and must be between 200-599";
      }
      return `- ${field} accepts ${operators} operator${
        config.operators.length > 1 ? "s" : ""
      }${constraints}`;
    })
    .join("\n");
  return `You are an expert at converting natural language queries into filters, understanding context and inferring filter types from natural expressions. Handle complex, ambiguous queries by breaking them down into clear filters. For status codes, use 200,400,500 variants - the application handles status grouping. Use ${usersReferenceMS} timestamp for time-related queries.

Examples:

# Complex Host Patterns
Query: "show requests from api.staging.company.com and test.company.com"
Result: [
  {
    field: "host",
    filters: [
      { operator: "is", value: "api.staging.company.com" },
      { operator: "is", value: "test.company.com" }
    ]
  }
]

Query: "localhost and 127.0.0.1 requests"
Result: [
  {
    field: "host",
    filters: [
      { operator: "is", value: "localhost" },
      { operator: "is", value: "127.0.0.1" }
    ]
  }
]

# Complex Path Patterns
Query: "find /api/v2/users/{userId}/profile and /api/v2/users/search requests"
Result: [
  {
    field: "paths",
    filters: [
      { operator: "startsWith", value: "api/v2/users" }
    ]
  }
]

Query: "/v1/api and /api/v1 timeouts"
Result: [
  {
    field: "paths",
    filters: [
      { operator: "startsWith", value: "v1/api" },
      { operator: "startsWith", value: "api/v1" }
    ]
  },
  {
    field: "status",
    filters: [{ operator: "is", value: 500 }]
  }
]

# Time Range Edge Cases
Query: "errors from last 30m and last 24h"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: 400 },
      { operator: "is", value: 500 }
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

# Status Code Variations
Query: "show me timeouts and server errors"
Result: [
  {
    field: "status",
    filters: [{ operator: "is", value: 500 }]
  }
]

Query: "client errors and failed requests"
Result: [
  {
    field: "status",
    filters: [{ operator: "is", value: 400 }]
  }
]

Query: "show successful and not found requests"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: 200 },
      { operator: "is", value: 404 }
    ]
  }
]

# Method Combinations
Query: "get, post and delete requests"
Result: [
  {
    field: "methods",
    filters: [
      { operator: "is", value: "GET" },
      { operator: "is", value: "POST" },
      { operator: "is", value: "DELETE" }
    ]
  }
]

Query: "read and write operations to /api"
Result: [
  {
    field: "methods",
    filters: [
      { operator: "is", value: "GET" },
      { operator: "is", value: "POST" }
    ]
  },
  {
    field: "paths",
    filters: [{ operator: "startsWith", value: "api" }]
  }
]

# Complex Combinations
Query: "failed requests to /v1/users or /v2/users from api.prod.com in last 2h"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: 400 },
      { operator: "is", value: 500 }
    ]
  },
  {
    field: "paths",
    filters: [
      { operator: "startsWith", value: "v1/users" },
      { operator: "startsWith", value: "v2/users" }
    ]
  },
  {
    field: "host",
    filters: [{ operator: "is", value: "api.prod.com" }]
  },
  {
    field: "startTime",
    filters: [{
      operator: "is",
      value: ${usersReferenceMS - 2 * 60 * 60 * 1000}
    }]
  }
]

Query: "localhost GET and POST /api errors since 2h and 30m ago"
Result: [
  {
    field: "host",
    filters: [{ operator: "is", value: "localhost" }]
  },
  {
    field: "methods",
    filters: [
      { operator: "is", value: "GET" },
      { operator: "is", value: "POST" }
    ]
  },
  {
    field: "paths",
    filters: [{ operator: "startsWith", value: "api" }]
  },
  {
    field: "status",
    filters: [
      { operator: "is", value: 400 },
      { operator: "is", value: 500 }
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

Remember:
${operatorsByField}
- since and endTime accept is operator for filtering logs by time range
- For status codes, use ONLY:
  • 200 for successful responses
  • 400 for client errors (4XX series)
  • 500 for server errors (5XX series)
- For relative time queries, support any combination of:
  • Nx[m] for minutes (e.g., 30m, 45m)
  • Nx[h] for hours (e.g., 1h, 24h)
  • Nx[d] for days (e.g., 1d, 7d)
  Multiple units can be combined (e.g., "1d 6h")

Special handling rules:
1. Always normalize paths by removing leading slash
2. For multiple time ranges, use the longest duration
3. For ambiguous terms like "failed", include both 400 and 500 status codes
4. Treat "read" operations as GET and "write" operations as POST
5. When seeing path parameters (e.g., {id}), match the base path
6. For IP addresses and localhost, treat them as host values

Error Handling Rules:
1. Invalid time formats: Convert to nearest supported range (e.g., "1w" → "7d")
2. Unknown HTTP methods: Default to GET for read-like terms, POST for write-like terms
3. Invalid status codes: Map to nearest category (e.g., 418 → 400, 503 → 500)
4. Malformed paths: Strip special characters and normalize

Ambiguity Resolution Priority:
1. Explicit over implicit (e.g., "GET" over "read-like" terms)
2. Specific over general (e.g., "/api/v2" over "/api")
3. Time ranges: Use most specific when multiple are valid
4. Status codes: When ambiguous between success/error, prefer error for terms like "failed", "issues"
5. Methods: When ambiguous between read/write, prefer read (GET) for safety

Output Validation:
1. Required fields must be present: field, filters
2. Filters must have: operator, value
3. Values must match field constraints:
   - methods: must be valid HTTP method
   - status: must be 200, 400, or 500
   - paths: must be normalized string
   - host: must be valid hostname/IP
   - time: must be valid timestamp or duration

Additional Examples:

# Error Handling Examples
Query: "show requests from last week"
Result: [
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "7d"  // Converts unsupported "week" to "7d"
    }]
  }
]

Query: "find browse requests to /api"
Result: [
  {
    field: "methods",
    filters: [{
      operator: "is",
      value: "GET"  // Maps "browse" to GET as read-like term
    }]
  },
  {
    field: "paths",
    filters: [{
      operator: "startsWith",
      value: "api"  // Normalized path
    }]
  }
]

# Ambiguity Resolution Examples
Query: "show api issues from last 2d and 6h"
Result: [
  {
    field: "paths",
    filters: [{
      operator: "startsWith",
      value: "api"
    }]
  },
  {
    field: "status",
    filters: [
      { operator: "is", value: 400 },
      { operator: "is", value: 500 }
    ]
  },
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "2d"  // Uses longest time range
    }]
  }
]

Query: "fetch requests to /api/v1 and /api"
Result: [
  {
    field: "methods",
    filters: [{
      operator: "is",
      value: "GET"  // "fetch" mapped to GET
    }]
  },
  {
    field: "paths",
    filters: [
      { operator: "startsWith", value: "api/v1" },  // More specific path first
      { operator: "startsWith", value: "api" }
    ]
  }
]`;
};
