import type { inferAsyncReturnType } from "@trpc/server";
import type { FetchCreateContextFnOptions } from "@trpc/server/adapters/fetch";

import { getAuth } from "../auth/get-auth";
import { db } from "../db";

export async function createContext({ req }: FetchCreateContextFnOptions) {
  // biome-ignore lint/suspicious/noExplicitAny:This has to be generic so any is okay
  let authResult = await getAuth(req as any);
  const { userId, orgId } = authResult;

  let ws: Awaited<ReturnType<typeof db.query.workspaces.findFirst>> = undefined;

  // Only attempt workspace query if we have both userId and orgId
  // This prevents unnecessary queries during auth setup phase
  if (orgId && userId) {
    try {
      ws = await db.query.workspaces.findFirst({
        where: (table, { eq, and, isNull }) =>
          and(eq(table.orgId, orgId), isNull(table.deletedAtM)),
      });

      // If workspace not found but we have valid auth context,
      // this might be post-login session synchronization issue for existing users
      if (!ws) {
        console.debug(
          "Workspace not found on first attempt - checking for post-login sync issue:",
          {
            orgId,
            userId,
            hasValidAuth: !!(userId && orgId),
          },
        );

        // For existing users logging in, add longer delay for session synchronization
        // This handles the case where auth cookies are set but context isn't fully synced
        await new Promise((resolve) => setTimeout(resolve, 200));

        // Retry auth validation to ensure session is fully synchronized
        try {
          // biome-ignore lint/suspicious/noExplicitAny:This has to be generic so any is okay
          const retryAuthResult = await getAuth(req as any);
          if (retryAuthResult.orgId && retryAuthResult.orgId !== orgId) {
            console.debug("Auth context changed after delay, using updated orgId:", {
              originalOrgId: orgId,
              updatedOrgId: retryAuthResult.orgId,
            });
            authResult = retryAuthResult;
          }
        } catch (authError) {
          console.debug("Auth retry failed, continuing with original:", authError);
        }

        // Try workspace query again with potentially updated auth context
        ws = await db.query.workspaces.findFirst({
          where: (table, { eq, and, isNull }) =>
            and(eq(table.orgId, authResult.orgId || orgId), isNull(table.deletedAtM)),
        });

        if (!ws) {
          console.debug(
            "Workspace still not found after post-login sync retry - may need frontend retry",
          );
        }
      }
    } catch (error) {
      // Log workspace query errors but don't fail context creation
      // This allows the frontend to handle workspace loading gracefully
      console.debug("Workspace query failed in context creation:", {
        orgId,
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
    tenant: authResult.orgId
      ? {
          id: authResult.orgId,
          role: authResult.role,
        }
      : null,
  };
}

export type Context = inferAsyncReturnType<typeof createContext>;
