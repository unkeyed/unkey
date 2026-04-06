import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRuntimeSettings, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

export type StringMatchMode = "exact" | "prefix" | "regex";

// ── Match expressions (protojson shape) ─────────────────────────────────

export type StringMatch = {
  ignoreCase?: boolean;
} & ({ exact: string } | { prefix: string } | { regex: string });

export type MatchExpr =
  | { path: { path: StringMatch } }
  | { method: { methods: string[] } }
  | { header: { name: string } & ({ present: boolean } | { value: StringMatch }) }
  | { queryParam: { name: string } & ({ present: boolean } | { value: StringMatch }) };

// ── Form-level match condition (ergonomic, converted to MatchExpr on submit) ─

export type MatchCondition =
  | { id: string; type: "path"; mode: StringMatchMode; value: string; ignoreCase?: boolean }
  | { id: string; type: "method"; methods: string[] }
  | {
      id: string;
      type: "header";
      name: string;
      present?: boolean;
      mode?: StringMatchMode;
      value?: string;
      ignoreCase?: boolean;
    }
  | {
      id: string;
      type: "queryParam";
      name: string;
      present?: boolean;
      mode?: StringMatchMode;
      value?: string;
      ignoreCase?: boolean;
    };

// ── Policy config types (protojson oneof shapes) ────────────────────────

export type KeyLocation =
  | { bearer: Record<string, never> }
  | { header: { name: string; stripPrefix?: string } }
  | { queryParam: { name: string } };

export type RateLimitKey =
  | { remoteIp: Record<string, never> }
  | { header: { name: string } }
  | { authenticatedSubject: Record<string, never> }
  | { path: Record<string, never> }
  | { principalClaim: { claimName: string } };

export type SentinelPolicy = {
  id: string;
  name: string;
  enabled: boolean;
  type: "keyauth" | "ratelimit" | "jwt" | "basicauth" | "iprules" | "openapi";
  keyauth?: {
    keySpaceIds: string[];
    locations?: KeyLocation[];
    permissionQuery?: string;
  };
  ratelimit?: {
    limit: number;
    windowMs: number;
    key?: RateLimitKey;
  };
  jwt?: { jwksUri?: string; issuer?: string; audience?: string[] };
  basicauth?: { credentials: { username: string; passwordHash: string }[] };
  iprules?: { allowlist: string[]; denylist: string[] };
  openapi?: { specPath: string };
  match?: MatchExpr[];
};

export type SentinelConfig = {
  policies: SentinelPolicy[];
};

// This is 100% not how we will do it later and is just a shortcut to use keyspace middleware before building the actual UI for it.
export const updateMiddleware = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      keyspaceIds: z.array(z.string()).max(10),
    }),
  )
  .mutation(async ({ ctx, input }) => {
    const sentinelConfig: SentinelConfig = {
      policies: [],
    };
    if (input.keyspaceIds.length > 0) {
      const keyspaces = await db.query.keyAuth
        .findMany({
          where: (table, { and, inArray }) =>
            and(inArray(table.id, input.keyspaceIds), eq(table.workspaceId, ctx.workspace.id)),
          columns: { id: true },
        })
        .catch((err) => {
          console.error(err);
          throw new TRPCError({
            code: "INTERNAL_SERVER_ERROR",
            message: "unable to load keyspaces",
          });
        });

      for (const id of input.keyspaceIds) {
        if (!keyspaces.find((ks) => ks.id === id)) {
          throw new TRPCError({
            code: "NOT_FOUND",
            message: `keyspace ${id} does not exist`,
          });
        }
      }

      sentinelConfig.policies.push({
        id: "keyauth-policy",
        name: "API Key Auth",
        enabled: true,
        type: "keyauth",
        keyauth: { keySpaceIds: keyspaces.map((ks) => ks.id) },
      });
    }

    const env = await db.query.environments.findFirst({
      where: and(
        eq(environments.id, input.environmentId),
        eq(environments.workspaceId, ctx.workspace.id),
      ),
      columns: { appId: true },
    });
    if (!env) {
      throw new TRPCError({ code: "NOT_FOUND", message: "Environment not found" });
    }

    const serialized = JSON.stringify(sentinelConfig);

    await db
      .insert(appRuntimeSettings)
      .values({
        workspaceId: ctx.workspace.id,
        appId: env.appId,
        environmentId: input.environmentId,
        sentinelConfig: serialized,
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      .onDuplicateKeyUpdate({ set: { sentinelConfig: serialized, updatedAt: Date.now() } });
  });
