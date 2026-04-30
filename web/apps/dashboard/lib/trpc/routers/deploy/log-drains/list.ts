import { and, db, eq, isNull } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { logDrainState, logDrains } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../trpc";

// Per-drain shape returned to the dashboard. Credentials never appear here:
// the encryption sits on log_drain_credentials and never leaves the
// coordinator process. Same for OAuth refresh tokens. The dashboard only
// ever needs to know whether the drain is healthy and what it is pointing
// at.
const drainOutputSchema = z.object({
  id: z.string(),
  workspaceId: z.string(),
  projectId: z.string().nullable(),
  name: z.string(),
  provider: z.enum(["axiom"]),
  config: z.unknown(),
  sources: z.array(z.string()),
  environments: z.array(z.string()),
  apps: z.array(z.string()),
  filters: z.unknown(),
  enabled: z.boolean(),
  createdAt: z.number(),
  updatedAt: z.number().nullable(),
  lastDeliveryAt: z.number().nullable(),
  lastError: z.string().nullable(),
  consecutiveFailures: z.number(),
  pausedReason: z.string().nullable(),
  totalRecordsDelivered: z.number(),
});

export const listLogDrains = workspaceProcedure
  .input(
    z.object({
      // Either project-scoped (drains belonging to a single project) or
      // workspace-scoped (drains with project_id=NULL plus, optionally,
      // every project drain in the workspace if the caller wants the
      // catch-all view). v1 ships project-scoped only; we accept the
      // discriminator now so the workspace settings page lands without a
      // breaking change.
      scope: z.enum(["project", "workspace"]),
      projectId: z.string().optional(),
    }),
  )
  .query(async ({ ctx, input }) => {
    // Narrow the project scope into a single, refinement-friendly variable
    // so the tRPC handler does not lean on a non-null assertion later.
    let projectFilter: ReturnType<typeof eq> | ReturnType<typeof isNull>;
    if (input.scope === "project") {
      if (!input.projectId) {
        throw new TRPCError({
          code: "BAD_REQUEST",
          message: "projectId is required for scope=project",
        });
      }
      projectFilter = eq(logDrains.projectId, input.projectId);
    } else {
      projectFilter = isNull(logDrains.projectId);
    }

    try {
      const where = and(
        eq(logDrains.workspaceId, ctx.workspace.id as string),
        isNull(logDrains.deletedAt),
        projectFilter,
      );

      // Single query joins the per-batch state row so the dashboard list
      // shows last-delivery + consecutive_failures without an N+1.
      const rows = await db
        .select({
          id: logDrains.id,
          workspaceId: logDrains.workspaceId,
          projectId: logDrains.projectId,
          name: logDrains.name,
          provider: logDrains.provider,
          config: logDrains.config,
          sources: logDrains.sources,
          environments: logDrains.environments,
          apps: logDrains.apps,
          filters: logDrains.filters,
          enabled: logDrains.enabled,
          createdAt: logDrains.createdAt,
          updatedAt: logDrains.updatedAt,
          lastDeliveryAt: logDrainState.lastDeliveryAt,
          lastError: logDrainState.lastError,
          consecutiveFailures: logDrainState.consecutiveFailures,
          pausedReason: logDrainState.pausedReason,
          totalRecordsDelivered: logDrainState.totalRecordsDelivered,
        })
        .from(logDrains)
        .leftJoin(logDrainState, eq(logDrainState.drainId, logDrains.id))
        .where(where);

      return rows.map((r) => ({
        id: r.id,
        workspaceId: r.workspaceId,
        projectId: r.projectId,
        name: r.name,
        provider: r.provider,
        config: r.config,
        sources: r.sources as string[],
        environments: r.environments as string[],
        apps: r.apps as string[],
        filters: r.filters,
        enabled: r.enabled,
        createdAt: r.createdAt,
        updatedAt: r.updatedAt ?? null,
        lastDeliveryAt: r.lastDeliveryAt ?? null,
        lastError: r.lastError ?? null,
        consecutiveFailures: r.consecutiveFailures ?? 0,
        pausedReason: r.pausedReason ?? null,
        totalRecordsDelivered: r.totalRecordsDelivered ?? 0,
      })) satisfies z.infer<typeof drainOutputSchema>[];
    } catch (error) {
      if (error instanceof TRPCError) {
        throw error;
      }
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to list log drains",
      });
    }
  });

export type LogDrainOutput = z.infer<typeof drainOutputSchema>;
