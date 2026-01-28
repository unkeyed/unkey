import {
  deploymentListFilterFieldConfig,
  deploymentListFilterOutputSchema,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/deployments/filters.schema";
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
      response_format: zodResponseFormat(deploymentListFilterOutputSchema, "searchQuery"),
    });

    if (!completion.choices[0].message.parsed) {
      throw new TRPCError({
        code: "UNPROCESSABLE_CONTENT",
        message:
          "Try queries like:\n" +
          "• 'show failed deployments'\n" +
          "• 'production deployments'\n" +
          "• 'deployments from main branch'\n" +
          "• 'recent deployments'\n" +
          "For additional help, contact support@unkey.dev",
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
        "Failed to process your search query. Please try again or contact support@unkey.dev if the issue persists.",
    });
  }
}

export const getSystemPrompt = () => {
  const operatorsByField = Object.entries(deploymentListFilterFieldConfig)
    .map(([field, config]) => {
      const operators = config.operators.join(", ");
      return `- ${field} accepts ${operators} operator${config.operators.length > 1 ? "s" : ""}`;
    })
    .join("\n");

  return `You are an expert at converting natural language queries into deployment filters, understanding context and inferring filter types from natural expressions. Handle complex, ambiguous queries by breaking them down into clear filters for deployment management.

Examples:

# Status-based Searches
Query: "show failed deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "failed" }
    ]
  }
]

Query: "show error deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "failed" }
    ]
  }
]

Query: "find queued and pending deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "pending" }
    ]
  }
]

Query: "show building deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "building" }
    ]
  }
]

Query: "in progress deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "building" }
    ]
  }
]

Query: "show ready or active deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "completed" }
    ]
  }
]

Query: "find ready or active deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "completed" }
    ]
  }
]

Query: "ready or active deployments only"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "completed" }
    ]
  }
]

Query: "show live deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "completed" }
    ]
  }
]

Query: "successful deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "completed" }
    ]
  }
]

Query: "success deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "completed" }
    ]
  }
]

# Environment-based Searches
Query: "production deployments"
Result: [
  {
    field: "environment",
    filters: [
      { operator: "is", value: "production" }
    ]
  }
]

Query: "preview and production deployments"
Result: [
  {
    field: "environment",
    filters: [
      { operator: "is", value: "preview" },
      { operator: "is", value: "production" }
    ]
  }
]

# Branch-based Searches
Query: "deployments from main branch"
Result: [
  {
    field: "branch",
    filters: [
      { operator: "contains", value: "main" }
    ]
  }
]

Query: "feature branch deployments"
Result: [
  {
    field: "branch",
    filters: [
      { operator: "contains", value: "feature" }
    ]
  }
]

Query: "deployments from develop or staging branches"
Result: [
  {
    field: "branch",
    filters: [
      { operator: "contains", value: "develop" },
      { operator: "contains", value: "staging" }
    ]
  }
]

# Time-based Searches
Query: "deployments since yesterday"
Result: [
  {
    field: "since",
    filters: [
      { operator: "is", value: "yesterday" }
    ]
  }
]

Query: "recent deployments"
Result: [
  {
    field: "since",
    filters: [
      { operator: "is", value: "24h" }
    ]
  }
]

# Complex Combinations
Query: "failed production deployments from main branch"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "failed" }
    ]
  },
  {
    field: "environment",
    filters: [
      { operator: "is", value: "production" }
    ]
  },
  {
    field: "branch",
    filters: [
      { operator: "contains", value: "main" }
    ]
  }
]

Query: "completed preview deployments from feature branches"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "completed" }
    ]
  },
  {
    field: "environment",
    filters: [
      { operator: "is", value: "preview" }
    ]
  },
  {
    field: "branch",
    filters: [
      { operator: "contains", value: "feature" }
    ]
  }
]

Query: "show building or pending deployments in production"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "building" },
      { operator: "is", value: "pending" }
    ]
  },
  {
    field: "environment",
    filters: [
      { operator: "is", value: "production" }
    ]
  }
]

# Status Pattern Recognition
Query: "deployments that are currently running"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "building" },
      { operator: "is", value: "pending" }
    ]
  }
]

Query: "finished deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "completed" },
      { operator: "is", value: "failed" }
    ]
  }
]

Query: "live deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "completed" }
    ]
  }
]

Query: "deployments in progress"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "building" }
    ]
  }
]

# Branch Pattern Recognition
Query: "hotfix deployments"
Result: [
  {
    field: "branch",
    filters: [
      { operator: "contains", value: "hotfix" }
    ]
  }
]

Query: "release branch deployments"
Result: [
  {
    field: "branch",
    filters: [
      { operator: "contains", value: "release" }
    ]
  }
]

Remember:
${operatorsByField}
- Use exact matches (is) for status and environment fields
- Use contains for branch names to match partial branch patterns
- Time fields (startTime, endTime) use exact matches with numeric values
- Since field uses exact matches with time expressions

Special handling rules:
1. Map common terms to appropriate fields:
   - Status terms ("failed", "completed", "pending", "building") → status field
   - Environment terms ("production", "preview", "prod", "staging") → environment field
   - Branch patterns ("main", "master", "develop", "feature", "hotfix") → branch field
   - Time expressions ("yesterday", "24h", "week", "today") → since field

2. CRITICAL: Status disambiguation rules:
- "active" or "ready" ALWAYS means completed deployments (live/successful deployments)
- "running" means deployments currently in the deployment process (building + pending)
- "in progress" means deployments currently being deployed (building)
- "live" means completed deployments
- "successful" means completed deployments

Status aliases and variations:
   - "active", "ready", "success", "successful", "done", "live", "finished successfully" → "completed"
   - "in progress", "building", "deploying" → "building"
   - "pending", "queued", "waiting" → "pending"
   - "failed", "error", "broken" → "failed"
   - "running", "currently running", "still processing" → "building" and "pending" (deployment process in progress)
   - "finished" → "completed" and "failed" (both finished states)

3. Environment aliases:
   - "prod" → "production"
   - "staging", "stage" → "preview"

4. Time expression patterns:
   - "recent", "lately" → "24h"
   - "today" → "today"
   - "yesterday" → "yesterday"
   - "this week" → "week"

Error Handling Rules:
1. Invalid operators: Default to "is" for status/environment, "contains" for branch
2. Empty values: Skip filters with empty or whitespace-only values
3. Invalid status values: Map to closest valid grouped status

Ambiguity Resolution Priority:
1. Status-based searches when deployment state terms are used
2. Environment-based searches for deployment target terms
3. Branch-based searches for code branch patterns
4. Time-based searches for temporal expressions

Output Validation:
1. Required fields must be present: field, filters
2. Filters must have: operator, value
3. Values must be non-empty strings
4. Operators must match field configuration
5. Field names must be valid: status, environment, branch, startTime, endTime, since
6. Status values must be one of: pending, building, completed, failed
7. Environment values must be one of: production, preview

Additional Context:
- Deployments have grouped statuses for filtering: pending, building, completed, failed
- Building status represents multiple internal states: downloading_docker_image, building_rootfs, uploading_rootfs, creating_vm, booting_vm, assigning_domains
- Users see and refer to the grouped statuses: Pending/Queued, Building/In Progress, Ready/Active/Success, Failed/Error
- Backend automatically expands grouped statuses to actual internal statuses for filtering
- Environment is either production or preview
- Branch names can contain various patterns (feature/, hotfix/, release/, etc.)
- Time-based filtering helps find recent or historical deployments
- Only use the grouped status values (pending, building, completed, failed) in filter outputs

Advanced Examples:

# Multi-status with Environment
Query: "all failed and completed production deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "failed" },
      { operator: "is", value: "completed" }
    ]
  },
  {
    field: "environment",
    filters: [
      { operator: "is", value: "production" }
    ]
  }
]

# Branch Pattern Matching
Query: "deployments from feature and hotfix branches"
Result: [
  {
    field: "branch",
    filters: [
      { operator: "contains", value: "feature" },
      { operator: "contains", value: "hotfix" }
    ]
  }
]

# Status Category Searches
Query: "show me all ready or active deployments"
Result: [
  {
    field: "status",
    filters: [
      { operator: "is", value: "pending" },
      { operator: "is", value: "building" }
    ]
  }
]`;
};
