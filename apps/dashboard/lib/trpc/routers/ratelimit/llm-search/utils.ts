import {
  filterOutputSchema,
  ratelimitFilterFieldConfig,
} from "@/app/(app)/[workspaceSlug]/ratelimits/[namespaceId]/logs/filters.schema";
import { TRPCError } from "@trpc/server";
import type OpenAI from "openai";
import { zodResponseFormat } from "openai/helpers/zod.mjs";

/**
 * Creates a Zod schema for validating LLM-generated structured filter output.
 * Used with OpenAI's parse completion to enforce type safety and validation rules
 * defined in FilterFieldConfigs.
 *
 *
 * @example
 * const schema = createFilterOutputSchema(
 *   z.enum(["status", "identifiers"]),
 *   z.enum(["is", "contains"]),
 *   ratelimitFilterFieldConfig
 * );
 *
 * const llmResponse = await openai.beta.chat.completions.parse({
 *   response_format: zodResponseFormat(schema, "searchQuery")
 * });
 */
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
      temperature: 0.2, // Range 0-2, lower = more focused/deterministic
      top_p: 0.1, // Alternative to temperature, controls randomness
      frequency_penalty: 0.5, // Range -2 to 2, higher = less repetition
      presence_penalty: 0.5, // Range -2 to 2, higher = more topic diversity
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
          "• 'show blocked requests'\n" +
          "• 'find requests from last 30 minutes'\n" +
          "• 'show requests containing test'\n" +
          "• 'find request req_abc123'\n" +
          "• 'show passed requests since 1h'\n" +
          "For additional help, contact support@unkey.dev",
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
  const operatorsByField = Object.entries(ratelimitFilterFieldConfig)
    .map(([field, config]) => {
      const operators = config.operators.join(", ");
      let constraints = "";
      if (field === "status") {
        constraints = ` and must be one of: "blocked", "passed"`;
      }
      return `- ${field} accepts ${operators} operator${
        config.operators.length > 1 ? "s" : ""
      }${constraints}`;
    })
    .join("\n");
  return `You are an expert at converting natural language queries into filters, understanding context and inferring filter types from natural expressions. Handle complex, ambiguous queries by breaking them down into clear filters. For status, use "blocked" or "passed". Use ${usersReferenceMS} timestamp for time-related queries.

Examples:

# Time Range Examples
Query: "show requests from last 30m"
Result: [
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "30m"
    }]
  }
]

Query: "find requests between yesterday and today"
Result: [
  {
    field: "startTime",
    filters: [{
      operator: "is",
      value: ${usersReferenceMS - 24 * 60 * 60 * 1000}
    }]
  },
  {
    field: "endTime",
    filters: [{
      operator: "is",
      value: ${usersReferenceMS}
    }]
  }
]

# Status Examples
Query: "show blocked requests"
Result: [
  {
    field: "status",
    filters: [{ operator: "is", value: "blocked" }]
  }
]

Query: "find all passed and blocked requests"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "passed" },
      { operator: "is", value: "blocked" }
    ]
  }
]

# Identifier Examples
Query: "find requests containing test"
Result: [
  {
    field: "identifiers",
    filters: [{ operator: "contains", value: "test" }]
  }
]

Query: "show requests with identifier abc-123"
Result: [
  {
    field: "identifiers",
    filters: [{ operator: "is", value: "abc-123" }]
  }
]

# Request ID Examples
Query: "find request req_123"
Result: [
  {
    field: "requestIds",
    filters: [{ operator: "is", value: "req_123" }]
  }
]

# Complex Combinations
Query: "show blocked requests from last 2h with identifier containing test"
Result: [
  {
    field: "status",
    filters: [{ operator: "is", value: "blocked" }]
  },
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "2h"
    }]
  },
  {
    field: "identifiers",
    filters: [{ operator: "contains", value: "test" }]
  }
]

Remember:
${operatorsByField}
- For relative time queries, support any combination of:
  • Nx[m] for minutes (e.g., 30m, 45m)
  • Nx[h] for hours (e.g., 1h, 24h)
  • Nx[d] for days (e.g., 1d, 7d)
  • Nx[d] for weeks (e.g., 1w, 2w)
  Multiple units can be combined (e.g., "1d 6h")

Special handling rules:
1. For multiple time ranges, use the longest duration
2. Status must be exactly "blocked" or "passed"
3. Identifiers support both exact matches and contains operations
4. Request IDs must be exact matches

Error Handling Rules:
1. Invalid status values: Default to "blocked" for negative terms (failed, error), "passed" for positive terms
2. For ambiguous identifiers, prefer "contains" over exact match

Ambiguity Resolution Priority:
1. Explicit over implicit (e.g., exact identifier over partial match)
2. Time ranges: Use most specific when multiple are valid
3. Status: When ambiguous, prefer explicit status values

Output Validation:
1. Required fields must be present: field, filters
2. Filters must have: operator, value
3. Values must match field constraints:
   - status: must be "blocked" or "passed"
   - time: must be valid timestamp or duration
   - identifiers and requestIds: must be strings

Additional Examples:

# Error Handling Examples
Query: "show requests from last week"
Result: [
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "1w"
    }]
  }
]

Query: "find failed requests"
Result: [
  {
    field: "status",
    filters: [{
      operator: "is",
      value: "blocked"  // Maps "failed" to blocked status
    }]
  }
]`;
};
