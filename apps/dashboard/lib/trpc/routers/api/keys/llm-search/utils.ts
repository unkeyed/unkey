import {
  filterOutputSchema,
  keysOverviewFilterFieldConfig,
} from "@/app/(app)/apis/[apiId]/_overview/filters.schema";
import { TRPCError } from "@trpc/server";
import { KEY_VERIFICATION_OUTCOMES } from "@unkey/clickhouse/src/keys/keys";
import type OpenAI from "openai";
import { zodResponseFormat } from "openai/helpers/zod.mjs";

/**
 * Creates a Zod schema for validating LLM-generated structured filter output for keys.
 * Used with OpenAI's parse completion to enforce type safety and validation rules
 * defined in keysOverviewFilterFieldConfig.
 *
 * @param openai - OpenAI client instance
 * @param userSearchMsg - User's natural language search query
 * @param usersReferenceMS - Reference timestamp in milliseconds
 * @returns Parsed structured search filters or null if OpenAI client is not available
 */
export async function getKeysStructuredSearchFromLLM(
  openai: OpenAI | null,
  userSearchMsg: string,
  usersReferenceMS: number,
) {
  try {
    if (!openai) {
      return null; // Skip LLM processing in development environment when OpenAI API key is not configured
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
          content: getKeysSystemPrompt(usersReferenceMS),
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
          "• 'show keys created in the last 30 minutes'\n" +
          "• 'find keys with name containing test'\n" +
          "• 'find key with id key_abc123'\n" +
          "• 'show successful keys since 1h'\n" +
          "• 'show keys with the outcome valid'\n" +
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

    if ((error as any).response?.status === 429) {
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

/**
 * Generates the system prompt for the key search LLM
 *
 * @param usersReferenceMS - Reference timestamp in milliseconds
 * @returns System prompt for the OpenAI conversation
 */
export const getKeysSystemPrompt = (usersReferenceMS: number) => {
  const operatorsByField = Object.entries(keysOverviewFilterFieldConfig)
    .map(([field, config]) => {
      const operators = config.operators.join(", ");
      let constraints = "";
      if (field === "outcomes") {
        constraints = ` and must be one of: ${KEY_VERIFICATION_OUTCOMES.map(
          (outcome) => `"${outcome}"`,
        ).join(", ")}`;
      }
      return `- ${field} accepts ${operators} operator${
        config.operators.length > 1 ? "s" : ""
      }${constraints}`;
    })
    .join("\n");
  return `You are an expert at converting natural language queries into filters for API key searches, understanding context and inferring filter types from natural expressions. Handle complex, ambiguous queries by breaking them down into clear filters. For outcomes, use one of the valid outcome values like "valid", "invalid", etc. Use ${usersReferenceMS} timestamp for time-related queries.

Examples:

# Time Range Examples
Query: "show keys from last 30m"
Result: [
  {
    field: "since",
    filters: [{ 
      operator: "is", 
      value: "30m"
    }]
  }
]

Query: "find keys between yesterday and today"
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

# Outcome Examples
Query: "show valid keys"
Result: [
  {
    field: "outcomes",
    filters: [{ operator: "is", value: "valid" }]
  }
]

Query: "find all invalid and expired keys"
Result: [
  {
    field: "outcomes",
    filters: [
      { operator: "is", value: "invalid" },
      { operator: "is", value: "expired" }
    ]
  }
]

# Key ID Examples
Query: "find key key_123"
Result: [
  {
    field: "keyIds",
    filters: [{ operator: "is", value: "key_123" }]
  }
]

Query: "show keys with id containing abc"
Result: [
  {
    field: "keyIds",
    filters: [{ operator: "contains", value: "abc" }]
  }
]

# Name Examples
Query: "find keys with name test"
Result: [
  {
    field: "names",
    filters: [{ operator: "is", value: "test" }]
  }
]

Query: "show keys with name containing prod"
Result: [
  {
    field: "names",
    filters: [{ operator: "contains", value: "prod" }]
  }
]

# Complex Combinations
Query: "show expired keys from last 2h with name containing test"
Result: [
  {
    field: "outcomes",
    filters: [{ operator: "is", value: "expired" }]
  },
  {
    field: "since",
    filters: [{ 
      operator: "is", 
      value: "2h"
    }]
  },
  {
    field: "names",
    filters: [{ operator: "contains", value: "test" }]
  }
]

Remember:
${operatorsByField}
- For relative time queries, support any combination of:
  • Nx[m] for minutes (e.g., 30m, 45m)
  • Nx[h] for hours (e.g., 1h, 24h)
  • Nx[d] for days (e.g., 1d, 7d)
  • Nx[w] for weeks (e.g., 1w, 2w)
  Multiple units can be combined (e.g., "1d 6h")

Special handling rules:
1. For multiple time ranges, use the longest duration
2. Outcomes must be one of the valid KEY_VERIFICATION_OUTCOMES values
3. Key IDs and names support both exact matches and contains operations

Error Handling Rules:
1. Invalid outcome values: Default to "invalid" for negative terms (failed, error), "valid" for positive terms
2. For ambiguous key names or IDs, prefer "contains" over exact match

Ambiguity Resolution Priority:
1. Explicit over implicit (e.g., exact name over partial match)
2. Time ranges: Use most specific when multiple are valid
3. Outcomes: When ambiguous, prefer explicit outcome values
4. External ID: When you see "External ID", treat it as "identities"

Output Validation:
1. Required fields must be present: field, filters
2. Filters must have: operator, value
3. Values must match field constraints:
   - outcomes: must be one of the valid outcomes
   - time: must be valid timestamp or duration
   - keyIds and names: must be strings

Additional Examples:

# Error Handling Examples
Query: "show keys from last week"
Result: [
  {
    field: "since",
    filters: [{ 
      operator: "is", 
      value: "1w"  
    }]
  }
]

Query: "find failed keys"
Result: [
  {
    field: "outcomes",
    filters: [{ 
      operator: "is", 
      value: "invalid"  // Maps "failed" to invalid status
    }]
  }
]`;
};
