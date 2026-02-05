import {
  runtimeLogsFilterFieldConfig,
  runtimeLogsFilterOutputSchema,
} from "@/lib/schemas/runtime-logs.filter.schema";
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
          name: "runtime-logs-ai-search",
          strict: true,
          schema: z.toJSONSchema(runtimeLogsFilterOutputSchema, { target: "draft-7" }),
        },
      },
    });

    if (!completion.choices[0].message.parsed) {
      throw new TRPCError({
        code: "UNPROCESSABLE_CONTENT",
        message:
          "Try phrases like:\n" +
          "• 'errors in the last hour'\n" +
          "• 'warnings containing timeout'\n" +
          "• 'debug logs from yesterday'\n" +
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
  const operatorsByField = Object.entries(runtimeLogsFilterFieldConfig)
    .map(([field, config]) => {
      const operators = config.operators.join(", ");
      let constraints = "";
      if (field === "severity") {
        constraints = " and must be one of: ERROR, WARN, INFO, DEBUG";
      }
      return `- ${field} accepts ${operators} operator${
        config.operators.length > 1 ? "s" : ""
      }${constraints}`;
    })
    .join("\n");

  return `You are an expert at converting natural language queries into filters for runtime container logs. Handle complex queries by breaking them into clear filters. Use ${usersReferenceMS} timestamp for time-related queries.

Examples:

# Severity Filtering
Query: "show errors in the last hour"
Result: [
  {
    field: "severity",
    filters: [
      { operator: "is", value: "ERROR" }
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

Query: "warnings and errors from yesterday"
Result: [
  {
    field: "severity",
    filters: [
      { operator: "is", value: "WARN" },
      { operator: "is", value: "ERROR" }
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

# Message Filtering
Query: "show warnings containing 'timeout'"
Result: [
  {
    field: "severity",
    filters: [
      { operator: "is", value: "WARN" }
    ]
  },
  {
    field: "message",
    filters: [
      { operator: "contains", value: "timeout" }
    ]
  }
]

Query: "find logs with deployment failed"
Result: [
  {
    field: "message",
    filters: [
      { operator: "contains", value: "deployment failed" }
    ]
  }
]

# Time Range Filtering
Query: "show all debug logs from yesterday"
Result: [
  {
    field: "severity",
    filters: [
      { operator: "is", value: "DEBUG" }
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

Query: "errors in last 30 minutes"
Result: [
  {
    field: "severity",
    filters: [
      { operator: "is", value: "ERROR" }
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

# Complex Combinations
Query: "find INFO logs with connection from last 2 hours"
Result: [
  {
    field: "severity",
    filters: [
      { operator: "is", value: "INFO" }
    ]
  },
  {
    field: "message",
    filters: [
      { operator: "contains", value: "connection" }
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

Query: "warnings and errors containing crash or panic since 6h"
Result: [
  {
    field: "severity",
    filters: [
      { operator: "is", value: "WARN" },
      { operator: "is", value: "ERROR" }
    ]
  },
  {
    field: "message",
    filters: [
      { operator: "contains", value: "crash" },
      { operator: "contains", value: "panic" }
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

Remember:
${operatorsByField}
- For relative time queries, support:
  • Nx[m] for minutes (e.g., 30m, 45m)
  • Nx[h] for hours (e.g., 1h, 24h)
  • Nx[d] for days (e.g., 1d, 7d)
- severity must be exactly one of: ERROR, WARN, INFO, DEBUG (case-sensitive)
- message operator "contains" for substring matching, "is" for exact match
- since and startTime/endTime are mutually exclusive - prefer since for relative time
- For multiple time ranges mentioned, use the longest duration

Special handling rules:
1. Map "error", "errors" → severity: ERROR
2. Map "warning", "warnings", "warn" → severity: WARN
3. Map "info", "information" → severity: INFO
4. Map "debug" → severity: DEBUG
5. For queries like "yesterday", use "1d" for since
6. For "last hour", use "1h" for since
7. For message filtering, extract quoted strings or key terms
8. When seeing "failed", "failure", "crash", "panic" → use contains on message field

Error Handling Rules:
1. Invalid time formats: Convert to nearest supported range (e.g., "1w" → "7d")
2. Unknown severity levels: Map to closest match or default to INFO
3. For multiple time ranges: Use the longest
4. For ambiguous terms like "issues" or "problems", include WARN and ERROR

Ambiguity Resolution:
1. Explicit severity over implicit (e.g., "ERROR" over "failed")
2. Specific time over general (e.g., "30m" over "recently")
3. When both severity and message filtering apply, include both
4. For terms like "critical", "severe" → map to ERROR
5. For terms like "notice", "alert" → map to WARN

Output Validation:
1. Required fields: field, filters
2. Filters must have: operator, value
3. Values must match field constraints:
   - severity: must be ERROR, WARN, INFO, or DEBUG
   - message: any string
   - since: valid duration string (e.g., "1h", "30m", "2d")
   - startTime/endTime: valid timestamp in milliseconds

Additional Examples:

# Error Handling
Query: "show logs from last week"
Result: [
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "7d"
    }]
  }
]

Query: "critical errors in production"
Result: [
  {
    field: "severity",
    filters: [{
      operator: "is",
      value: "ERROR"
    }]
  },
  {
    field: "message",
    filters: [{
      operator: "contains",
      value: "production"
    }]
  }
]

# Ambiguity Resolution
Query: "show issues from last day"
Result: [
  {
    field: "severity",
    filters: [
      { operator: "is", value: "WARN" },
      { operator: "is", value: "ERROR" }
    ]
  },
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "1d"
    }]
  }
]`;
};
