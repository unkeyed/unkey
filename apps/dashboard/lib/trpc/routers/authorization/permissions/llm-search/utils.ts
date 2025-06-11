import {
  filterOutputSchema,
  permissionsFilterFieldConfig,
} from "@/app/(app)/authorization/permissions/filters.schema";
import { TRPCError } from "@trpc/server";
import type OpenAI from "openai";
import { zodResponseFormat } from "openai/helpers/zod.mjs";

export async function getStructuredSearchFromLLM(openai: OpenAI | null, userSearchMsg: string) {
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
          content: getSystemPrompt(),
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
          "• 'find roles with admin permissions'\n" +
          "• 'show roles containing api.read'\n" +
          "• 'find roles assigned to user keys'\n" +
          "• 'show roles with database permissions'\n" +
          "• 'find all admin and moderator roles'\n" +
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

export const getSystemPrompt = () => {
  const operatorsByField = Object.entries(permissionsFilterFieldConfig)
    .map(([field, config]) => {
      const operators = config.operators.join(", ");
      return `- ${field}: ${operators}`;
    })
    .join("\n");

  return `Convert natural language queries into permission filters. Use context to infer the correct field and operator.

FIELD OPERATORS:
${operatorsByField}

OPERATOR RULES:
- "is": exact matches (IDs, specific slugs like "api.read")
- "contains": partial matches (names, descriptions, general terms)
- "startsWith/endsWith": prefix/suffix patterns

EXAMPLES:

Query: "admin permissions"
→ [{"field": "name", "filters": [{"operator": "contains", "value": "admin"}]}]

Query: "api.read permission"
→ [{"field": "slug", "filters": [{"operator": "is", "value": "api.read"}]}]

Query: "permissions for database access"
→ [{"field": "description", "filters": [{"operator": "contains", "value": "database"}]}]

Query: "admin permissions with database access"
→ [
  {"field": "name", "filters": [{"operator": "contains", "value": "admin"}]},
  {"field": "description", "filters": [{"operator": "contains", "value": "database"}]}
]

Query: "permissions assigned to admin role"
→ [{"field": "roleName", "filters": [{"operator": "contains", "value": "admin"}]}]

Query: "permissions starting with api_"
→ [{"field": "name", "filters": [{"operator": "startsWith", "value": "api_"}]}]

PRIORITY RULES:
1. Technical terms (api.read, role_123) → use "is" with slug/roleId
2. Descriptive terms (admin, database) → use "contains" with name/description
3. When ambiguous, search multiple relevant fields
4. Normalize plurals to singular, lowercase technical terms

OUTPUT: Always return valid filters with field, operator, and non-empty value.`;
};
