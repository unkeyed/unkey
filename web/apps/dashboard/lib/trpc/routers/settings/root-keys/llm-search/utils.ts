import {
  filterOutputSchema,
  rootKeysFilterFieldConfig,
} from "@/app/(app)/[workspaceSlug]/settings/root-keys/filters.schema";
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
          "• 'find keys starting with sk_'\n" +
          "• 'show keys with delete permissions'\n" +
          "• 'find production keys'\n" +
          "• 'show keys named admin'\n" +
          "• 'find keys with api.create permissions'\n" +
          "For additional help, contact support@unkey.com",
      });
    }

    return completion.choices[0].message.parsed;
  } catch (error) {
    console.error(
      `Something went wrong when querying OpenAI. Input: ${JSON.stringify(
        userSearchMsg,
      )}\n Output ${(error as Error).message}`,
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

export const getSystemPrompt = () => {
  const operatorsByField = Object.entries(rootKeysFilterFieldConfig)
    .map(([field, config]) => {
      const operators = config.operators.join(", ");
      return `- ${field} accepts ${operators} operator${config.operators.length > 1 ? "s" : ""}`;
    })
    .join("\n");

  return `You are an expert at converting natural language queries into root key filters, understanding context and inferring filter types from natural expressions. Handle complex, ambiguous queries by breaking them down into clear filters for root key management.

Examples:

# Key Name Patterns
Query: "find admin and production keys"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "admin" },
      { operator: "contains", value: "production" }
    ]
  }
]

Query: "show keys named exactly 'master_key'"
Result: [
  {
    field: "name",
    filters: [
      { operator: "is", value: "master_key" }
    ]
  }
]

Query: "find development and staging keys"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "development" },
      { operator: "contains", value: "staging" }
    ]
  }
]

# Key Start/Prefix Patterns
Query: "keys starting with sk_"
Result: [
  {
    field: "start",
    filters: [
      { operator: "contains", value: "sk_" }
    ]
  }
]

Query: "find keys with prefix rk_prod"
Result: [
  {
    field: "start",
    filters: [
      { operator: "contains", value: "rk_prod" }
    ]
  }
]

Query: "show all sk_ and rk_ keys"
Result: [
  {
    field: "start",
    filters: [
      { operator: "contains", value: "sk_" },
      { operator: "contains", value: "rk_" }
    ]
  }
]

Query: "keys with exact start sk_1234"
Result: [
  {
    field: "start",
    filters: [
      { operator: "is", value: "sk_1234" }
    ]
  }
]

# Permission-based Searches
Query: "keys with delete permissions"
Result: [
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "delete" }
    ]
  }
]

Query: "find keys with api.create and api.delete"
Result: [
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "api.create" },
      { operator: "contains", value: "api.delete" }
    ]
  }
]



Query: "show keys with rbac permissions"
Result: [
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "rbac" }
    ]
  }
]

Query: "find keys with ratelimit and identity permissions"
Result: [
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "ratelimit" },
      { operator: "contains", value: "identity" }
    ]
  }
]

# Complex Combinations
Query: "admin keys starting with sk_ with delete permissions"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "admin" }
    ]
  },
  {
    field: "start",
    filters: [
      { operator: "contains", value: "sk_" }
    ]
  },
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "delete" }
    ]
  }
]

Query: "production or staging keys with api permissions"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "production" },
      { operator: "contains", value: "staging" }
    ]
  },
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "api" }
    ]
  }
]

Query: "find keys named 'master' with sk_ prefix and create permissions"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "master" }
    ]
  },
  {
    field: "start",
    filters: [
      { operator: "contains", value: "sk_" }
    ]
  },
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "create" }
    ]
  }
]

# Critical Permission Searches
Query: "keys with critical permissions"
Result: [
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "delete" },
      { operator: "contains", value: "decrypt" },
      { operator: "contains", value: "remove" }
    ]
  }
]

Query: "dangerous keys"
Result: [
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "delete" },
      { operator: "contains", value: "decrypt" }
    ]
  }
]

# Environment-based Searches
Query: "development keys"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "dev" },
      { operator: "contains", value: "development" }
    ]
  }
]

Query: "prod keys with write access"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "prod" },
      { operator: "contains", value: "production" }
    ]
  },
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "write" },
      { operator: "contains", value: "create" },
      { operator: "contains", value: "update" }
    ]
  }
]

# Service-specific Searches
Query: "api keys"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "api" }
    ]
  },
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "api" }
    ]
  }
]

Query: "keys for user management"
Result: [
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "identity" },
      { operator: "contains", value: "user" }
    ]
  }
]

Query: "ratelimit service keys"
Result: [
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "ratelimit" }
    ]
  }
]

Remember:
${operatorsByField}
- Use exact matches (is) for specific key names, exact prefixes
- Use contains for partial matches within names, prefixes, or permission patterns
- For key searches, consider both name and start fields for comprehensive results
- For permission searches, match against the full permission string (e.g., "api.*.create_key")

Special handling rules:
1. Map common terms to appropriate fields:
   - Environment terms ("prod", "dev", "staging") → typically name field
   - Prefix patterns ("sk_", "rk_") → start field
   - Permission patterns ("delete", "create", "api.read") → permission field
   - Service names ("api", "rbac", "ratelimit") → could be name or permission
2. When terms could apply to multiple fields, search both:
   - "api" → check both name and permission fields
   - "admin" → check both name and permission fields
3. Handle variations and aliases:
   - "prod" → "production"
   - "dev" → "development"
   - "critical" → "delete", "decrypt", "remove"
   - "dangerous" → "delete", "decrypt"

Error Handling Rules:
1. Invalid operators: Default to "contains" for ambiguous searches
2. Empty values: Skip filters with empty or whitespace-only values
3. Conflicting constraints: Use the most specific constraint

Ambiguity Resolution Priority:
1. Exact matches over partial when context suggests precision
2. Permission-based searches when security terms are used
3. Environment-based searches for deployment terms
4. Multiple field searches when terms could apply to different contexts

Output Validation:
1. Required fields must be present: field, filters
2. Filters must have: operator, value
3. Values must be non-empty strings
4. Operators must match field configuration
5. Field names must be valid: name, start, permission

Additional Context:
- Root keys are sensitive authentication tokens
- Key prefixes (start field) help identify key types and environments
- Permissions follow patterns like "api.*.create_key", "rbac.*.read_role"
- Critical permissions (delete, decrypt, remove) require special attention
- Keys are often named by environment (prod, dev, staging) or service (api, admin)

Advanced Examples:

# Multi-environment Searches
Query: "all production and staging admin keys"
Result: [
  {
    field: "name",
    filters: [
      { operator: "contains", value: "production" },
      { operator: "contains", value: "staging" },
      { operator: "contains", value: "admin" }
    ]
  }
]

# Permission Category Searches
Query: "keys with any write permissions"
Result: [
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "create" },
      { operator: "contains", value: "update" },
      { operator: "contains", value: "write" },
      { operator: "contains", value: "delete" }
    ]
  }
]

# Exclusion Patterns (use positive filtering)
Query: "non-admin keys"
Result: [
  {
    field: "permission",
    filters: [
      { operator: "contains", value: "read" }
    ]
  }
]`;
};
