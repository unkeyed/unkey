import {
  filterOutputSchema,
  rolesFilterFieldConfig,
} from "@/app/(app)/[workspace]/authorization/roles/filters.schema";
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
  const operatorsByField = Object.entries(rolesFilterFieldConfig)
    .map(([field, config]) => {
      const operators = config.operators.join(", ");
      return `- ${field} accepts ${operators} operator${config.operators.length > 1 ? "s" : ""}`;
    })
    .join("\n");

  return `You are an expert at converting natural language queries into role filters, understanding context and inferring filter types from natural expressions. Handle complex, ambiguous queries by breaking them down into clear filters for role management.

Examples:

# Role Name Patterns
Query: "find admin and moderator roles"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "admin" },
      { operator: "contains", value: "moderator" }
    ]
  }
]

Query: "show roles starting with user_"
Result: [
  {
    field: "name",
    filters: [
      { operator: "startsWith", value: "user_" }
    ]
  }
]

# Role Description Patterns
Query: "roles for database access"
Result: [
  {
    field: "description",
    filters: [
      { operator: "contains", value: "database" }
    ]
  }
]

Query: "find roles with API permissions in description"
Result: [
  {
    field: "description",
    filters: [
      { operator: "contains", value: "API" }
    ]
  }
]

# Permission-based Searches
Query: "roles with api.read and api.write permissions"
Result: [
  {
    field: "permissionSlug",
    filters: [
      { operator: "is", value: "api.read" },
      { operator: "is", value: "api.write" }
    ]
  }
]

Query: "find roles containing database permissions"
Result: [
  {
    field: "permissionName",
    filters: [
      { operator: "contains", value: "database" }
    ]
  }
]

Query: "show roles with admin permissions"
Result: [
  {
    field: "permissionName",
    filters: [
      { operator: "contains", value: "admin" }
    ]
  }
]

# Key-based Searches
Query: "roles assigned to user keys"
Result: [
  {
    field: "keyName",
    filters: [
      { operator: "contains", value: "user" }
    ]
  }
]

Query: "find roles with api_key_123"
Result: [
  {
    field: "keyId",
    filters: [
      { operator: "is", value: "api_key_123" }
    ]
  }
]

Query: "roles for production keys"
Result: [
  {
    field: "keyName",
    filters: [
      { operator: "contains", value: "production" }
    ]
  }
]

# Complex Combinations
Query: "admin roles with database permissions and user keys"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "admin" }
    ]
  },
  {
    field: "permissionName",
    filters: [
      { operator: "contains", value: "database" }
    ]
  },
  {
    field: "keyName",
    filters: [
      { operator: "contains", value: "user" }
    ]
  }
]

Query: "find moderator or admin roles with api permissions"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "moderator" },
      { operator: "contains", value: "admin" }
    ]
  },
  {
    field: "permissionName",
    filters: [
      { operator: "contains", value: "api" }
    ]
  }
]

Query: "roles named exactly 'super_admin' with write permissions"
Result: [
  {
    field: "name",
    filters: [
      { operator: "is", value: "super_admin" }
    ]
  },
  {
    field: "permissionSlug",
    filters: [
      { operator: "contains", value: "write" }
    ]
  }
]

# Specific Permission Slug Searches
Query: "roles with user.create and user.delete permissions"
Result: [
  {
    field: "permissionSlug",
    filters: [
      { operator: "is", value: "user.create" },
      { operator: "is", value: "user.delete" }
    ]
  }
]

Query: "find roles ending with _admin"
Result: [
  {
    field: "name",
    filters: [
      { operator: "endsWith", value: "_admin" }
    ]
  }
]

# Key ID Searches
Query: "roles for key starting with key_"
Result: [
  {
    field: "keyId",
    filters: [
      { operator: "startsWith", value: "key_" }
    ]
  }
]

Remember:
${operatorsByField}
- Use exact matches (is) for specific role names, permission slugs, or key IDs
- Use contains for partial matches within names, descriptions, or permissions
- Use startsWith/endsWith for prefix/suffix matching
- For role searches, consider both name and description fields
- For permission searches, distinguish between permissionName (human-readable) and permissionSlug (technical identifier)
- For key searches, distinguish between keyName (human-readable) and keyId (technical identifier)

Special handling rules:
1. Map common terms to appropriate fields:
   - "admin", "moderator", "user" → typically name or permissionName
   - Permission patterns like "api.read", "user.create" → permissionSlug
   - Technical IDs starting with prefixes → keyId or exact matches
2. When terms could apply to multiple fields, prioritize:
   - Exact technical terms (slugs, IDs) → use "is" operator
   - Descriptive terms → use "contains" operator
   - Role hierarchies → check both name and permissions
3. Handle plurals and variations:
   - "admins" → "admin" (normalize to singular)
   - "APIs" → "api" (normalize case and plurals)

Error Handling Rules:
1. Invalid operators: Default to "contains" for ambiguous searches
2. Empty values: Skip filters with empty or whitespace-only values
3. Conflicting constraints: Use the most specific constraint

Ambiguity Resolution Priority:
1. Exact matches over partial (e.g., role name "admin" vs description containing "admin")
2. Technical identifiers over human-readable names when context suggests precision
3. Permission-based searches over role names when permissions are explicitly mentioned
4. Multiple field searches when terms could apply to different contexts

Output Validation:
1. Required fields must be present: field, filters
2. Filters must have: operator, value
3. Values must be non-empty strings
4. Operators must match field configuration
5. Field names must be valid: name, description, permissionSlug, permissionName, keyId, keyName

Additional Examples:

# Error Handling Examples
Query: "show roles with empty permissions"
Result: [
  {
    field: "permissionName",
    filters: [{
      operator: "is",
      value: ""  // Handles empty/null permission searches
    }]
  }
]

Query: "find development and staging roles"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "development" },
      { operator: "contains", value: "staging" }
    ]
  }
]

# Ambiguity Resolution Examples
Query: "api roles"
Result: [
  {
    field: "name",
    filters: [{
      operator: "contains",
      value: "api"
    }]
  },
  {
    field: "permissionName",
    filters: [{
      operator: "contains",
      value: "api"
    }]
  }
]

Query: "user management permissions"
Result: [
  {
    field: "permissionName",
    filters: [{
      operator: "contains",
      value: "user"
    }]
  }
]`;
};
