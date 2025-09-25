import {
  auditFilterOutputSchema,
  auditLogsFilterFieldConfig,
} from "@/app/(app)/[workspaceSlug]/audit/filters.schema";
import { TRPCError } from "@trpc/server";
import { unkeyAuditLogEvents } from "@unkey/schema/src/auditlog";
import type OpenAI from "openai";
import { zodResponseFormat } from "openai/helpers/zod.mjs";

export async function getStructuredAuditSearchFromLLM(
  openai: OpenAI | null,
  userSearchMsg: string,
  usersReferenceMS: number,
) {
  try {
    // Skip LLM processing in development environment when OpenAI API key is not configured
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
          content: getAuditSystemPrompt(usersReferenceMS),
        },
        {
          role: "user",
          content: userSearchMsg,
        },
      ],
      response_format: zodResponseFormat(auditFilterOutputSchema, "searchQuery"),
    });

    if (!completion.choices[0].message.parsed) {
      throw new TRPCError({
        code: "UNPROCESSABLE_CONTENT",
        message:
          "Try using phrases like:\n" +
          "• 'show events from last 30 minutes'\n" +
          "• 'find activity for user user_abc123'\n" +
          "• 'show create_key events'\n" +
          "• 'find logs from bucket audit_xyz'\n" +
          "• 'show activity since 1h'\n" +
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

export const getAuditSystemPrompt = (usersReferenceMS: number) => {
  const validEventTypes = Object.values(unkeyAuditLogEvents.enum);
  const operatorsByField = Object.entries(auditLogsFilterFieldConfig)
    .map(([field, config]) => {
      const operators = config.operators.join(", ");
      return `- ${field} accepts ${operators} operator${config.operators.length > 1 ? "s" : ""}`;
    })
    .join("\n");

  return `You are an expert at converting natural language queries into audit log filters, understanding context and inferring filter types from natural expressions. Handle complex, ambiguous queries by breaking them down into clear filters. Use ${usersReferenceMS} timestamp for time-related queries.

You have access to a specific set of audit log event types in the Unkey system. When a user refers to an event, map it to the closest matching event from this set. For example, "set override" should be mapped to "ratelimit.set_override" and "create key" should be mapped to "key.create".

Examples:
# Time Range Examples
Query: "show events from last 30m"
Result: [
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "30m"
    }]
  }
]

Query: "find logs between yesterday and today"
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

# Event Type Examples
Query: "show key.create events"
Result: [
  {
    field: "events",
    filters: [{ operator: "is", value: "key.create" }]
  }
]

Query: "find all key.delete and ratelimit.delete_override events"
Result: [
  {
    field: "events",
    filters: [
      { operator: "is", value: "key.delete" },
      { operator: "is", value: "ratelimit.delete_override" }
    ]
  }
]

# User Examples
Query: "find events for user user_123"
Result: [
  {
    field: "users",
    filters: [{ operator: "is", value: "user_123" }]
  }
]

# Root Key Examples
Query: "show logs with root key root_abc"
Result: [
  {
    field: "rootKeys",
    filters: [{ operator: "is", value: "root_abc" }]
  }
]

# Bucket Examples
Query: "find logs in bucket audit_xyz"
Result: [
  {
    field: "bucket",
    filters: [{ operator: "is", value: "audit_xyz" }]
  }
]

# Complex Combinations
Query: "show key.create events from last 2h for user user_123"
Result: [
  {
    field: "events",
    filters: [{ operator: "is", value: "key.create" }]
  },
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "2h"
    }]
  },
  {
    field: "users",
    filters: [{ operator: "is", value: "user_123" }]
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
2. Events must match exactly one of the valid event types:
${validEventTypes.map((event) => `   - ${event}`).join("\n")}
3. When a user uses generic terms, map to the specific event type:
   - "create key" → "key.create"
   - "update key" → "key.update"
   - "delete key" → "key.delete"
   - "set override" → "ratelimit.set_override"
   - workspace.create, workspace.update, workspace.delete, workspace.opt_in
   - gateway.create, llmGateway.create, llmGateway.delete
   - api.create, api.update, api.delete
   - key.create, key.update, key.delete, key.reroll
   - ratelimitNamespace.create, ratelimitNamespace.update, ratelimitNamespace.delete
   - vercelIntegration.create, vercelIntegration.update, vercelIntegration.delete
   - vercelBinding.create, vercelBinding.update, vercelBinding.delete
   - role.create, role.update, role.delete
   - permission.create, permission.update, permission.delete
   - authorization.connect_role_and_permission, authorization.disconnect_role_and_permissions
   - authorization.connect_role_and_key, authorization.disconnect_role_and_key
   - authorization.connect_permission_and_key, authorization.disconnect_permission_and_key
   - secret.create, secret.decrypt, secret.update
   - webhook.create, webhook.update, webhook.delete
   - reporter.create
   - identity.create, identity.update, identity.delete
   - ratelimit.create, ratelimit.update, ratelimit.delete
   - ratelimit.set_override, ratelimit.read_override, ratelimit.delete_override
   - auditLogBucket.create
3. Users should be matched exactly and typically follow patterns like user_xyz123
4. Root keys typically follow patterns like root_abc123
5. Buckets typically follow patterns like audit_xyz123

Ambiguity Resolution Priority:
1. Explicit over implicit (e.g., exact event type over partial match)
2. Time ranges: Use most specific when multiple are valid
3. When ambiguous, prefer exact matches

Output Validation:
1. Required fields must be present: field, filters
2. Filters must have: operator, value
3. Values must match field constraints:
   - events, users, rootKeys, bucket: must be strings
   - time: must be valid timestamp or duration

Additional Examples:
# Error Handling Examples
Query: "show events from last week"
Result: [
  {
    field: "since",
    filters: [{
      operator: "is",
      value: "1w"
    }]
  }
]`;
};
