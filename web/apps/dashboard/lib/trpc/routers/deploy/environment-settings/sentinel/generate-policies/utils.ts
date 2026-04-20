import {
  type MatchConditionFormValues,
  type PolicyFormValues,
  matchConditionSchema,
} from "@/app/(app)/[workspaceSlug]/projects/[projectId]/(overview)/sentinel-policies/components/add-panel/schema";
import { TRPCError } from "@trpc/server";
import type OpenAI from "openai";
import { z } from "zod";

const llmHttpMethodSchema = z.enum(["GET", "POST", "PUT", "PATCH", "DELETE", "HEAD", "OPTIONS"]);

// Flat schema, all fields required so OpenAI strict mode is happy.
// For path:        name is ignored, methods is empty.
// For method:      mode/value/name are ignored.
// For header/qp:   methods is empty.
const llmMatchConditionSchema = z.object({
  type: z.enum(["path", "method", "header", "queryParam"]),
  mode: z.enum(["exact", "prefix", "regex"]),
  value: z.string(),
  name: z.string(),
  methods: z.array(llmHttpMethodSchema),
});

// Flat schema, all fields required so OpenAI strict mode is happy.
// For ratelimit: locationType/locationName/permissionQuery/action are ignored.
// For keyauth:   limit/windowMs/identifierSource/identifierValue/action are ignored.
// For firewall:  limit/windowMs/identifierSource/identifierValue/locationType/locationName/permissionQuery are ignored.
const llmPolicySchema = z.object({
  name: z.string(),
  type: z.enum(["ratelimit", "keyauth", "firewall"]),
  matchConditions: z.array(llmMatchConditionSchema),
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

type LLMMatchCondition = z.infer<typeof llmMatchConditionSchema>;

function toFormMatchConditions(raw: LLMMatchCondition[]): MatchConditionFormValues[] {
  return raw
    .map((m): MatchConditionFormValues | null => {
      const id = crypto.randomUUID();
      if (m.type === "path") {
        return { id, type: "path", mode: m.mode, value: m.value };
      }
      if (m.type === "method") {
        return { id, type: "method", methods: m.methods };
      }
      if (m.type === "header") {
        return { id, type: "header", name: m.name, mode: m.mode, value: m.value };
      }
      return { id, type: "queryParam", name: m.name, mode: m.mode, value: m.value };
    })
    .filter((c): c is MatchConditionFormValues => {
      if (c === null) {
        return false;
      }
      return matchConditionSchema.safeParse(c).success;
    });
}

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
      const matchConditions = toFormMatchConditions(p.matchConditions);

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
          matchConditions,
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
          matchConditions,
          action: p.action,
        };
      }

      return {
        type: "ratelimit",
        name: p.name,
        environmentId: "__all__",
        matchConditions,
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

## Match conditions

Every policy has a "matchConditions" array that scopes it to requests matching ALL listed conditions (AND semantics). An empty array applies the policy to all traffic.

Condition shapes (flat schema, unused fields must still be present — use empty strings / empty arrays):
- path:       { type: "path",       mode: "exact"|"prefix"|"regex", value: "/some/path", name: "", methods: [] }
- method:     { type: "method",     mode: "exact", value: "", name: "", methods: ["GET", "POST", ...] }
- header:     { type: "header",     mode: "exact"|"prefix"|"regex", value: "abc", name: "X-Header", methods: [] }
- queryParam: { type: "queryParam", mode: "exact"|"prefix"|"regex", value: "abc", name: "param", methods: [] }

Choosing a path mode:
- "exact"  -> a single path, e.g. "/"
- "prefix" -> a subtree, e.g. "/api" matches "/api/foo"
- "regex"  -> alternations / patterns, e.g. "^/(admin|debug)"

Because conditions AND together, to match ANY of several paths use a SINGLE regex condition (not many path conditions). To express independent scopes, emit separate policies.

## Rules
- Return 1-5 policies
- Always include a matchConditions array (empty [] if the policy applies to all traffic)
- keyauth typically comes first (it authenticates the request; ratelimit can then use authenticatedSubject)
- For burst: short window (1s-10s), low limit
- For sustained: longer window (1min-1h), higher limit
- Per-key: identifierSource=authenticatedSubject
- Per-workspace: identifierSource=principalField, identifierValue=workspace_id
- Per-user: identifierSource=principalField, identifierValue=identity_id
- Firewall is used when the user wants to block/deny traffic (e.g. block a path, block unauthenticated requests)
- Scope policies with matchConditions whenever the user mentions a path, method, header, or query param

## Examples

Input: "authenticate with bearer token, rate limit 100/min per key"
Output: [
  { name: "Key Authentication", type: "keyauth", matchConditions: [], locationType: "bearer", locationName: "", permissionQuery: "", limit: 0, windowMs: 0, identifierSource: "remoteIp", identifierValue: "", action: "ACTION_DENY" },
  { name: "Per-Key Limit", type: "ratelimit", matchConditions: [], limit: 100, windowMs: 60000, identifierSource: "authenticatedSubject", identifierValue: "", locationType: "bearer", locationName: "", permissionQuery: "", action: "ACTION_DENY" }
]

Input: "keyauth via X-API-Key header with api:read permission, then burst 5/s sustained 200/min"
Output: [
  { name: "Key Authentication", type: "keyauth", matchConditions: [], locationType: "header", locationName: "X-API-Key", permissionQuery: "api:read", limit: 0, windowMs: 0, identifierSource: "remoteIp", identifierValue: "", action: "ACTION_DENY" },
  { name: "Burst Protection", type: "ratelimit", matchConditions: [], limit: 5, windowMs: 1000, identifierSource: "authenticatedSubject", identifierValue: "", locationType: "bearer", locationName: "", permissionQuery: "", action: "ACTION_DENY" },
  { name: "Sustained Limit", type: "ratelimit", matchConditions: [], limit: 200, windowMs: 60000, identifierSource: "authenticatedSubject", identifierValue: "", locationType: "bearer", locationName: "", permissionQuery: "", action: "ACTION_DENY" }
]

Input: "block all traffic to /admin"
Output: [
  { name: "Block Admin", type: "firewall", matchConditions: [{ type: "path", mode: "prefix", value: "/admin", name: "", methods: [] }], action: "ACTION_DENY", limit: 0, windowMs: 0, identifierSource: "remoteIp", identifierValue: "", locationType: "bearer", locationName: "", permissionQuery: "" }
]

Input: "block all wordpress scrape targets"
Output: [
  { name: "Block WordPress Probes", type: "firewall", matchConditions: [{ type: "path", mode: "regex", value: "^/(wp-admin|wp-login\\\\.php|xmlrpc\\\\.php|wp-content|wp-includes)", name: "", methods: [] }], action: "ACTION_DENY", limit: 0, windowMs: 0, identifierSource: "remoteIp", identifierValue: "", locationType: "bearer", locationName: "", permissionQuery: "" }
]

Input: "ratelimit 10/minute to /"
Output: [
  { name: "Rate Limit Root", type: "ratelimit", matchConditions: [{ type: "path", mode: "exact", value: "/", name: "", methods: [] }], limit: 10, windowMs: 60000, identifierSource: "remoteIp", identifierValue: "", locationType: "bearer", locationName: "", permissionQuery: "", action: "ACTION_DENY" }
]

Input: "per-key limit 20/min, per-workspace limit 100/min"
Output: [
  { name: "Per-Key Limit", type: "ratelimit", matchConditions: [], limit: 20, windowMs: 60000, identifierSource: "authenticatedSubject", identifierValue: "", locationType: "bearer", locationName: "", permissionQuery: "", action: "ACTION_DENY" },
  { name: "Per-Workspace Limit", type: "ratelimit", matchConditions: [], limit: 100, windowMs: 60000, identifierSource: "principalField", identifierValue: "workspace_id", locationType: "bearer", locationName: "", permissionQuery: "", action: "ACTION_DENY" }
]`;
}
