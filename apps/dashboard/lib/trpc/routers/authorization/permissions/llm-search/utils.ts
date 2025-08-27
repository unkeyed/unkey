import {
  filterOutputSchema,
  permissionsFilterFieldConfig,
} from "@/app/(app)/[workspace]/authorization/permissions/filters.schema";
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
      return `- ${field} accepts ${operators} operator${config.operators.length > 1 ? "s" : ""}`;
    })
    .join("\n");

  return `You are an expert at converting natural language queries into permission filters, understanding context and inferring filter types from natural expressions. Handle complex, ambiguous queries by breaking them down into clear filters for permission management.

Examples:

# Permission Name Patterns
Query: "find admin and user permissions"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "admin" },
      { operator: "contains", value: "user" }
    ]
  }
]

Query: "show permissions starting with api_"
Result: [
  {
    field: "name",
    filters: [
      { operator: "startsWith", value: "api_" }
    ]
  }
]

# Permission Description Patterns
Query: "permissions for database access"
Result: [
  {
    field: "description",
    filters: [
      { operator: "contains", value: "database" }
    ]
  }
]

Query: "find permissions with read access in description"
Result: [
  {
    field: "description",
    filters: [
      { operator: "contains", value: "read" }
    ]
  }
]

# Permission Slug Searches
Query: "permissions with api.read and api.write slugs"
Result: [
  {
    field: "slug",
    filters: [
      { operator: "is", value: "api.read" },
      { operator: "is", value: "api.write" }
    ]
  }
]

Query: "find permissions with slugs containing user"
Result: [
  {
    field: "slug",
    filters: [
      { operator: "contains", value: "user" }
    ]
  }
]

Query: "show permissions ending with .create"
Result: [
  {
    field: "slug",
    filters: [
      { operator: "endsWith", value: ".create" }
    ]
  }
]

# Role-based Permission Searches
Query: "permissions assigned to admin role"
Result: [
  {
    field: "roleName",
    filters: [
      { operator: "contains", value: "admin" }
    ]
  }
]

Query: "find permissions for role_123"
Result: [
  {
    field: "roleId",
    filters: [
      { operator: "is", value: "role_123" }
    ]
  }
]

Query: "permissions for moderator and editor roles"
Result: [
  {
    field: "roleName",
    filters: [
      { operator: "contains", value: "moderator" },
      { operator: "contains", value: "editor" }
    ]
  }
]

# Complex Combinations
Query: "admin permissions with database access"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "admin" }
    ]
  },
  {
    field: "description",
    filters: [
      { operator: "contains", value: "database" }
    ]
  }
]

Query: "find user.create or user.delete permissions assigned to admin roles"
Result: [
  {
    field: "slug",
    filters: [
      { operator: "is", value: "user.create" },
      { operator: "is", value: "user.delete" }
    ]
  },
  {
    field: "roleName",
    filters: [
      { operator: "contains", value: "admin" }
    ]
  }
]

Query: "permissions named exactly 'super_admin' for role starting with admin_"
Result: [
  {
    field: "name",
    filters: [
      { operator: "is", value: "super_admin" }
    ]
  },
  {
    field: "roleName",
    filters: [
      { operator: "startsWith", value: "admin_" }
    ]
  }
]

# Specific Permission Searches
Query: "permissions with api.read and api.write slugs"
Result: [
  {
    field: "slug",
    filters: [
      { operator: "is", value: "api.read" },
      { operator: "is", value: "api.write" }
    ]
  }
]

Query: "find permissions ending with _manage"
Result: [
  {
    field: "name",
    filters: [
      { operator: "endsWith", value: "_manage" }
    ]
  }
]

# Role ID Searches
Query: "permissions for roles starting with role_"
Result: [
  {
    field: "roleId",
    filters: [
      { operator: "startsWith", value: "role_" }
    ]
  }
]

Query: "permissions assigned to multiple specific roles"
Result: [
  {
    field: "roleId",
    filters: [
      { operator: "is", value: "role_123" },
      { operator: "is", value: "role_456" }
    ]
  }
]

Remember:
${operatorsByField}
- Use exact matches (is) for specific permission names, slugs, or role IDs
- Use contains for partial matches within names, descriptions, or role names
- Use startsWith/endsWith for prefix/suffix matching
- For permission searches, consider both name and description fields
- For slug searches, use exact matches for technical identifiers like "api.read"
- For role searches, distinguish between roleName (human-readable) and roleId (technical identifier)

Special handling rules:
1. When terms could apply to multiple fields, prioritize:
   - Exact technical terms (slugs, IDs) → use "is" operator
   - Descriptive terms → use "contains" operator
   - Permission hierarchies → check both name and description
2. Handle plurals and variations:
   - "admins" → "admin" (normalize to singular)
   - "APIs" → "api" (normalize case and plurals)

Error Handling Rules:
1. Invalid operators: Default to "contains" for ambiguous searches
2. Empty values: Skip filters with empty or whitespace-only values
3. Conflicting constraints: Use the most specific constraint

Ambiguity Resolution Priority:
1. Exact matches over partial (e.g., permission name "admin" vs description containing "admin")
2. Technical identifiers (slugs, role IDs) over human-readable names when context suggests precision
3. Permission-based searches over role names when permissions are explicitly mentioned
4. Multiple field searches when terms could apply to different contexts

Output Validation:
1. Required fields must be present: field, filters
2. Filters must have: operator, value
3. Values must be non-empty strings
4. Operators must match field configuration
5. Field names must be valid: name, description, slug, roleId, roleName

Additional Examples:

# Error Handling Examples
Query: "show permissions with empty descriptions"
Result: [
  {
    field: "description",
    filters: [{
      operator: "is",
      value: ""  // Handles empty/null description searches
    }]
  }
]

Query: "find read and write permissions"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "read" },
      { operator: "contains", value: "write" }
    ]
  }
]

# Ambiguity Resolution Examples
Query: "api permissions"
Result: [
  {
    field: "name",
    filters: [{
      operator: "contains",
      value: "api"
    }]
  },
  {
    field: "description",
    filters: [{
      operator: "contains",
      value: "api"
    }]
  }
]

Query: "user management permissions"
Result: [
  {
    field: "name",
    filters: [{
      operator: "contains",
      value: "user"
    }]
  },
  {
    field: "description",
    filters: [{
      operator: "contains",
      value: "management"
    }]
  }
]`;
};
