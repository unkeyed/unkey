import {
  domainPair,
  pairConfigForMode,
  wwwModeSchema,
} from "@/lib/collections/deploy/edge-redirects.schema";
import { and, db, eq, inArray } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { frontlineRoutes, projects } from "@unkey/db/src/schema";
import { z } from "zod";

/**
 * updateEdgeRedirects writes the joint www-handling state for the apex/
 * www pair that `domain` belongs to. The mutation always touches both
 * rows in a single transaction so toggling cleanly reverts whichever
 * direction was previously configured.
 *
 * If a side does not exist (the user only owns one of the two domains),
 * we silently skip writing it. The mode the user can pick is gated on
 * the get response's `*Exists` flags so this should not normally happen,
 * but we don't want to fail the whole save if the row vanishes between
 * read and write.
 */
export const updateEdgeRedirects = workspaceProcedure
  .use(withRatelimit(ratelimit.update))
  .input(
    z.object({
      projectId: z.string().min(1, "Project ID is required"),
      domain: z.string().min(1, "Domain is required"),
      wwwMode: wwwModeSchema,
    }),
  )
  .mutation(async ({ input, ctx }) => {
    const pair = domainPair(input.domain);
    const target = pairConfigForMode(input.wwwMode);

    const rows = await db
      .select({
        id: frontlineRoutes.id,
        fqdn: frontlineRoutes.fullyQualifiedDomainName,
      })
      .from(frontlineRoutes)
      .innerJoin(
        projects,
        and(eq(projects.id, frontlineRoutes.projectId), eq(projects.workspaceId, ctx.workspace.id)),
      )
      .where(
        and(
          eq(frontlineRoutes.projectId, input.projectId),
          inArray(frontlineRoutes.fullyQualifiedDomainName, [pair.apex, pair.www]),
        ),
      );

    if (rows.length === 0) {
      throw new TRPCError({
        code: "NOT_FOUND",
        message: "No active routes for this domain pair. Wait for verification to complete.",
      });
    }

    const apex = rows.find((r) => r.fqdn === pair.apex);
    const www = rows.find((r) => r.fqdn === pair.www);

    // The mode the user picked may require a row that does not exist
    // (e.g. "stripWww" needs the www row). Surface that as a clear error
    // rather than silently dropping the rule.
    if (input.wwwMode === "stripWww" && !www) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: `${pair.www} is not registered as a custom domain.`,
      });
    }
    if (input.wwwMode === "addWww" && !apex) {
      throw new TRPCError({
        code: "BAD_REQUEST",
        message: `${pair.apex} is not registered as a custom domain.`,
      });
    }

    const now = Date.now();
    await db.transaction(async (tx) => {
      if (apex) {
        await tx
          .update(frontlineRoutes)
          .set({ edgeRedirectConfig: JSON.stringify(target.apex), updatedAt: now })
          .where(eq(frontlineRoutes.id, apex.id));
      }
      if (www) {
        await tx
          .update(frontlineRoutes)
          .set({ edgeRedirectConfig: JSON.stringify(target.www), updatedAt: now })
          .where(eq(frontlineRoutes.id, www.id));
      }
    });

    return { success: true };
  });
