import {
  filterOutputSchema,
  keysListFilterFieldConfig,
} from "@/app/(app)/[workspaceSlug]/apis/[apiId]/keys/[keyAuthId]/_components/filters.schema";
import { TRPCError } from "@trpc/server";
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
export async function getKeysStructuredSearchFromLLM(openai: OpenAI | null, userSearchMsg: string) {
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
          content: getKeysSystemPrompt(),
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

    if (
      (
        error as {
          response?: {
            status: number;
          };
        }
      ).response?.status === 429
    ) {
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

/**
 * Generates the system prompt for the key search LLM
 *
 * @param usersReferenceMS - Reference timestamp in milliseconds
 * @returns System prompt for the OpenAI conversation
 */
export const getKeysSystemPrompt = () => {
  const operatorsByField = Object.entries(keysListFilterFieldConfig)
    .map(([field, config]) => {
      const operators =
        Array.isArray(config?.operators) && config.operators.length > 0
          ? config.operators.join(", ")
          : "no specific operators listed"; // Fallback message
      const operatorText = Array.isArray(config?.operators)
        ? `accepts ${operators} operator${config.operators.length > 1 ? "s" : ""}`
        : "has specific operator constraints"; // General fallback
      return `- ${field} (${config.type} type) ${operatorText}`;
    })
    .join("\n");

  return `You are an expert at converting natural language queries into structured JSON filters for API key searches. You understand context and can infer filter types and operators from natural expressions. Your goal is to generate a JSON array matching the required schema based *only* on the available fields and operators.

Available filter fields and their operators:
${operatorsByField}

Map user intent to the correct field and operator:
- 'is', 'equals', 'exact match' -> "is" operator
- 'contains', 'includes', 'has' -> "contains" operator
- 'starts with', 'prefix is' -> "startsWith" operator
- 'ends with', 'suffix is' -> "endsWith" operator
- If the query implies searching by external ID, user ID, or similar identifiers, use the "identities" field.

Examples:

# Key ID Examples
Query: "find key key_123abc456"
Result: [
  {
    "field": "keyIds",
    "filters": [{ "operator": "is", "value": "key_123abc456" }]
  }
]

Query: "show keys with id containing test"
Result: [
  {
    "field": "keyIds",
    "filters": [{ "operator": "contains", "value": "test" }]
  }
]

Query: "find keys starting with key_abc"
Result: [
  {
    "field": "keyIds",
    "filters": [{ "operator": "startsWith", "value": "key_abc" }]
  }
]

Query: "keys ending in xyz"
Result: [
  {
    "field": "keyIds",
    "filters": [{ "operator": "endsWith", "value": "xyz" }]
  }
]


# Name Examples
Query: "find keys with name Production Key"
Result: [
  {
    "field": "names",
    "filters": [{ "operator": "is", "value": "Production Key" }]
  }
]

Query: "show keys with name containing 'prod'"
Result: [
  {
    "field": "names",
    "filters": [{ "operator": "contains", "value": "prod" }]
  }
]

Query: "find key names that start with 'test-'"
Result: [
  {
    "field": "names",
    "filters": [{ "operator": "startsWith", "value": "test-" }]
  }
]

# Identities (External ID / Owner ID) Examples
Query: "find keys for identity user_123"
Result: [
  {
    "field": "identities",
    "filters": [{ "operator": "is", "value": "user_123" }]
  }
]

Query: "show keys associated with external id containing @example.com"
Result: [
  {
    "field": "identities",
    "filters": [{ "operator": "contains", "value": "@example.com" }]
  }
]

Query: "find keys for user IDs starting with 'org_'"
Result: [
  {
    "field": "identities",
    "filters": [{ "operator": "startsWith", "value": "org_" }]
  }
]

# Complex Combinations
Query: "show keys with name containing 'staging' and identity 'dev_user'"
Result: [
  {
    "field": "names",
    "filters": [{ "operator": "contains", "value": "staging" }]
  },
  {
    "field": "identities",
    "filters": [{ "operator": "is", "value": "dev_user" }]
  }
]

Query: "find keys starting with 'temp_' or ending with '_test' for identity containing 'customer'"
Result: [
 {
    "field": "keyIds",
    "filters": [
        { "operator": "startsWith", "value": "temp_" },
        { "operator": "endsWith", "value": "_test" }
     ]
  },
  {
    "field": "identities",
    "filters": [{ "operator": "contains", "value": "customer" }]
  }
]


Important Rules:
1.  Only use the fields: keyIds, names, identities.
2.  Only use the operators specified for each field: is, contains, startsWith, endsWith.
3.  Interpret "External ID", "User ID", "Owner ID" or similar concepts as the "identities" field.
4.  If multiple conditions are given for the same field (e.g., "name contains test or name contains debug"), create multiple filter objects within the 'filters' array for that field.
5.  If a query is ambiguous (e.g., "find test keys"), prefer the 'contains' operator for the most likely field (e.g., 'names' or 'keyIds'). If unsure, ask for clarification (though you must output JSON, so make a best guess).
6.  The output MUST be a valid JSON array conforming to the schema, even if the query is unclear or doesn't map perfectly. Output an empty array [] if no filters can be reliably determined.

Output Validation Requirements:
- The root structure must be a JSON array.
- Each element in the array must be an object with "field" (string) and "filters" (array) properties.
- The "field" value must be one of: "keyIds", "names", "identities".
- Each object within the "filters" array must have "operator" (string) and "value" (string) properties.
- The "operator" value must be one of: "is", "contains", "startsWith", "endsWith".
- The "value" must be a string.
`;
};
