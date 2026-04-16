import type { PolicyFormValues } from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/sentinel-policies/components/add-panel/schema";
import { TRPCError } from "@trpc/server";
import type OpenAI from "openai";
import { z } from "zod";

// Flat schema — all fields required so OpenAI strict mode is happy.
// For ratelimit: locationType/locationName/permissionQuery/action are ignored.
// For keyauth:   limit/windowMs/identifierSource/identifierValue/action are ignored.
// For firewall:  limit/windowMs/identifierSource/identifierValue/locationType/locationName/permissionQuery are ignored.
const llmPolicySchema = z.object({
  name: z.string(),
  type: z.enum(["ratelimit", "keyauth", "firewall"]),
  // ratelimit fields
  limit: z.number().int().min(0),
  windowMs: z.number().int().min(0),
  identifierSource: z.enum([
    "remoteIp",
    "authenticatedSubject",
    "principalField",
    "header",
    "path",
  ]),
  identifierValue: z.string(),
  // keyauth fields
  locationType: z.enum(["bearer", "header", "queryParam"]),
  locationName: z.string(),
  permissionQuery: z.string(),
  // firewall fields
  action: z.enum(["ACTION_DENY"]),
});

const llmOutputSchema = z.object({
  policies: z.array(llmPolicySchema),
});

export async function generatePoliciesFromLLM(
  openai: OpenAI | null,
  query: string,
): Promise<PolicyFormValues[]> {
  if (!openai) {
    throw new TRPCError({
      code: "PRECONDITION_FAILED",
      message: "OpenAI isn't configured correctly, please check your API key",
    });
  }

  try {
    const completion = await openai.chat.completions.parse({
      model: "gpt-4o-mini",
      temperature: 0.2,
      top_p: 0.1,
      frequency_penalty: 0.5,
      presence_penalty: 0.5,
      n: 1,
      messages: [
        { role: "system", content: getSystemPrompt() },
        { role: "user", content: query },
      ],
      response_format: {
        type: "json_schema",
        json_schema: {
          name: "sentinel-policies",
          strict: true,
          schema: z.toJSONSchema(llmOutputSchema, { target: "draft-7" }),
        },
      },
    });

    const parsed = completion.choices[0].message.parsed;
    if (!parsed) {
      throw new TRPCError({
        code: "UNPROCESSABLE_CONTENT",
        message:
          "Could not generate policies. Try describing your needs, e.g.:\n" +
          "• 'authenticate via bearer token, then burst 10 req/s'\n" +
          "• 'keyauth with permission check, rate limit 100/min per key'\n" +
          "• 'per-key limit 20/min, per-workspace limit 100/min'",
      });
    }

    const validated = llmOutputSchema.parse(parsed);

    return validated.policies.map((p): PolicyFormValues => {
      if (p.type === "keyauth") {
        const location =
          p.locationType === "bearer"
            ? { id: crypto.randomUUID(), locationType: "bearer" as const }
            : p.locationType === "header"
              ? { id: crypto.randomUUID(), locationType: "header" as const, name: p.locationName }
              : {
                  id: crypto.randomUUID(),
                  locationType: "queryParam" as const,
                  name: p.locationName,
                };

        return {
          type: "keyauth",
          name: p.name,
          environmentId: "__all__",
          matchConditions: [],
          keySpaceIds: [],
          locations: [location],
          permissionQuery: p.permissionQuery,
        };
      }

      if (p.type === "firewall") {
        return {
          type: "firewall",
          name: p.name,
          environmentId: "__all__",
          matchConditions: [],
          action: p.action,
        };
      }

      return {
        type: "ratelimit",
        name: p.name,
        environmentId: "__all__",
        matchConditions: [],
        limit: p.limit,
        windowMs: p.windowMs,
        identifierSource: p.identifierSource,
        identifierValue: p.identifierValue,
      };
    });
  } catch (error) {
    console.error(
      `Failed to generate policies. Input: ${JSON.stringify(query)}\nError: ${(error as Error).message}`,
    );

    if (error instanceof TRPCError) {
      throw error;
    }

    if ((error as { response: { status: number } }).response?.status === 429) {
      throw new TRPCError({
        code: "TOO_MANY_REQUESTS",
        message: "Rate limit exceeded. Please try again in a few minutes.",
      });
    }

    throw new TRPCError({
      code: "INTERNAL_SERVER_ERROR",
      message: "Failed to generate policies. Please try again.",
    });
  }
}

function getSystemPrompt(): string {
  return `You generate sentinel policy configurations (keyauth, ratelimit, and firewall) from natural language descriptions.

## Policy types

### keyauth -- authenticates requests via API key
- locationType: "bearer" (Authorization: Bearer <key>), "header" (custom header), "queryParam" (query param)
- locationName: header name or param name (empty for bearer)
- permissionQuery: optional permission filter like "api:read" (usually empty)
- Set limit=0, windowMs=0, identifierSource="remoteIp", identifierValue="", action="ACTION_DENY" for keyauth policies

### ratelimit -- limits request rate
- identifierSource: remoteIp | authenticatedSubject | principalField | header | path
- identifierValue: header name or field path (workspace_id, identity_id, plan...); empty for others
- windowMs: 1000=1s, 5000=5s, 60000=1min, 300000=5min, 3600000=1h
- Set locationType="bearer", locationName="", permissionQuery="", action="ACTION_DENY" for ratelimit policies

### firewall -- blocks requests matching conditions
- action: "ACTION_DENY" (deny matching requests with 403)
- Set limit=0, windowMs=0, identifierSource="remoteIp", identifierValue="" for firewall policies
- Set locationType="bearer", locationName="", permissionQuery="" for firewall policies

## Rules
- Return 1-5 policies
- keyauth typically comes first (it authenticates the request; ratelimit can then use authenticatedSubject)
- For burst: short window (1s-10s), low limit
- For sustained: longer window (1min-1h), higher limit
- Per-key: identifierSource=authenticatedSubject
- Per-workspace: identifierSource=principalField, identifierValue=workspace_id
- Per-user: identifierSource=principalField, identifierValue=identity_id
- Firewall is used when the user wants to block/deny traffic (e.g. block a path, block unauthenticated requests)

## Examples

Input: "authenticate with bearer token, rate limit 100/min per key"
Output: [
  { name: "Key Authentication", type: "keyauth", locationType: "bearer", locationName: "", permissionQuery: "", limit: 0, windowMs: 0, identifierSource: "remoteIp", identifierValue: "", action: "ACTION_DENY" },
  { name: "Per-Key Limit", type: "ratelimit", limit: 100, windowMs: 60000, identifierSource: "authenticatedSubject", identifierValue: "", locationType: "bearer", locationName: "", permissionQuery: "", action: "ACTION_DENY" }
]

Input: "keyauth via X-API-Key header with api:read permission, then burst 5/s sustained 200/min"
Output: [
  { name: "Key Authentication", type: "keyauth", locationType: "header", locationName: "X-API-Key", permissionQuery: "api:read", limit: 0, windowMs: 0, identifierSource: "remoteIp", identifierValue: "", action: "ACTION_DENY" },
  { name: "Burst Protection", type: "ratelimit", limit: 5, windowMs: 1000, identifierSource: "authenticatedSubject", identifierValue: "", locationType: "bearer", locationName: "", permissionQuery: "", action: "ACTION_DENY" },
  { name: "Sustained Limit", type: "ratelimit", limit: 200, windowMs: 60000, identifierSource: "authenticatedSubject", identifierValue: "", locationType: "bearer", locationName: "", permissionQuery: "", action: "ACTION_DENY" }
]

Input: "block all traffic to /admin"
Output: [
  { name: "Block Admin", type: "firewall", action: "ACTION_DENY", limit: 0, windowMs: 0, identifierSource: "remoteIp", identifierValue: "", locationType: "bearer", locationName: "", permissionQuery: "" }
]

Input: "per-key limit 20/min, per-workspace limit 100/min"
Output: [
  { name: "Per-Key Limit", type: "ratelimit", limit: 20, windowMs: 60000, identifierSource: "authenticatedSubject", identifierValue: "", locationType: "bearer", locationName: "", permissionQuery: "", action: "ACTION_DENY" },
  { name: "Per-Workspace Limit", type: "ratelimit", limit: 100, windowMs: 60000, identifierSource: "principalField", identifierValue: "workspace_id", locationType: "bearer", locationName: "", permissionQuery: "", action: "ACTION_DENY" }
]`;
}
