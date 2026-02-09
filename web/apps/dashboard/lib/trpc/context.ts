import type { inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";
import type { NextRequest } from "next/server";

import { env } from "../env";
import { getAuth } from "../auth/get-auth";
import { db, eq, schema } from "../db";

export async function createContext({ req }: FetchCreateContextFnOptions) {
  let authResult = await getAuth(req as NextRequest);
  const { userId, orgId } = authResult;
  const environment = env();

  // If user is authenticated but has no orgId (activeOrganizationId is null),
  // look up their org membership for Better Auth users
  let resolvedOrgId = orgId;
  let resolvedRole = authResult.role;
  
  if (!orgId && userId && environment.AUTH_PROVIDER === "better-auth") {
    try {
      const memberships = await db.query.baMember.findMany({
        where: eq(schema.baMember.userId, userId),
      });

      if (memberships.length === 1 && memberships[0].organizationId) {
        // Single org - use it
        resolvedOrgId = memberships[0].organizationId;
        resolvedRole = memberships[0].role;
        console.log("[TRPC Context] Resolved single org for user:", {
          userId,
          orgId: resolvedOrgId,
          role: resolvedRole,
        });
      } else if (memberships.length > 1) {
        // Multiple orgs - user needs to select one
        // Leave orgId as null, the proxy should have redirected them
        console.log("[TRPC Context] User has multiple orgs but no activeOrganizationId:", {
          userId,
          orgCount: memberships.length,
        });
      }
    } catch (error) {
      console.error("[TRPC Context] Error looking up user memberships:", error);
    }
  }

  let ws: Awaited<ReturnType<typeof db.query.workspaces.findFirst>> = undefined;

  // Only attempt workspace query if we have both userId and orgId
  // This prevents unnecessary queries during auth setup phase
  if (resolvedOrgId && userId) {
    try {
      ws = await db.query.workspaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.orgId, resolvedOrgId), isNull(table.deletedAtM)),
      });

      // If workspace not found but we have valid auth context,
      // this might be post-login session synchronization issue for existing users
      if (!ws) {
        console.log(
          "Workspace not found on first attempt, retrying after delay:",
          {
            orgId: resolvedOrgId,
          },
        );

        // For existing users logging in, add longer delay for session synchronization
        // This handles the case where auth cookies are set but context isn't fully synced
        await new Promise((resolve) => setTimeout(resolve, 200));

        // Retry auth validation to ensure session is fully synchronized
        try {
          const retryAuthResult = await getAuth(req as NextRequest);
          if (retryAuthResult.orgId && retryAuthResult.orgId !== resolvedOrgId) {
            console.log("Auth context changed after delay, using updated orgId:", {
              originalOrgId: resolvedOrgId,
              updatedOrgId: retryAuthResult.orgId,
            });
            authResult = retryAuthResult;
            resolvedOrgId = retryAuthResult.orgId;
            resolvedRole = retryAuthResult.role;
          }
        } catch (authError) {
          console.log("Auth retry failed, continuing with original:", authError);
        }

        // Try workspace query again with potentially updated auth context
        ws = await db.query.workspaces.findFirst({
          where: (table, { eq, and, isNull }) =>
            and(eq(table.orgId, resolvedOrgId), isNull(table.deletedAtM)),
        });

        if (!ws) {
          console.log(
            "No workspace found for tenant after retry:",
            resolvedOrgId,
          );
        }
      }
    } catch (error) {
      // Log workspace query errors but don't fail context creation
      // This allows the frontend to handle workspace loading gracefully
      console.log("Workspace query failed in context creation:", {
        orgId: resolvedOrgId,
        userId,
        error: error instanceof Error ? error.message : String(error),
      });
      ws = undefined;
    }
  }

  return {
    req,
    audit: {
      userAgent: req.headers.get("user-agent") ?? undefined,
      location: req.headers.get("x-forwarded-for") ?? process.env.VERCEL_REGION ?? "unknown",
    },
    user: authResult.userId ? { id: authResult.userId } : null,
    workspace: ws,
    tenant: resolvedOrgId
      ? {
          id: resolvedOrgId,
          role: resolvedRole,
        }
      : null,
  };
}

export type Context = inferAsyncReturnType<typeof createContext>;
