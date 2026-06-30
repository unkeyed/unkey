import { and, db, eq } from "@/lib/db";
import { TRPCError } from "@trpc/server";
import { appRuntimeSettings, environments } from "@unkey/db/src/schema";
import { z } from "zod";
import { workspaceProcedure } from "../../../../trpc";

// krane narrows the probe fields to int32 before handing them to Kubernetes.
const INT32_MAX = 2_147_483_647;

// The healthcheck is persisted as JSON, so an unbounded path would let an
// authenticated user bloat the row. No real probe path approaches this; it is
// an abuse bound, not a functional limit.
const MAX_HEALTHCHECK_PATH_LENGTH = 2048;

export const updateHealthcheck = workspaceProcedure
  .input(
    z.object({
      environmentId: z.string(),
      // krane casts these straight into an int32 corev1.Probe
      // (svc/krane/internal/deployment/apply.go), so the real upper bound is
      // int32 max — capping tighter would reject already-stored healthchecks
      // and make them uneditable. We only reject what Kubernetes itself
      // rejects: negatives, non-integers, and (for period/timeout/threshold)
      // zero. Initial delay may be 0.
      healthcheck: z
        .object({
          method: z.enum(["GET", "POST"]),
          path: z.string().trim().min(1, "Path is required").max(MAX_HEALTHCHECK_PATH_LENGTH),
          intervalSeconds: z.number().int().min(1).max(INT32_MAX).default(10),
          timeoutSeconds: z.number().int().min(1).max(INT32_MAX).default(5),
          failureThreshold: z.number().int().min(1).max(INT32_MAX).default(3),
          initialDelaySeconds: z.number().int().min(0).max(INT32_MAX).default(0),
        })
        .nullable(),
    }),
  )
  .mutation(async ({ ctx, input }) => {
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

    await db
      .insert(appRuntimeSettings)
      .values({
        workspaceId: ctx.workspace.id,
        appId: env.appId,
        environmentId: input.environmentId,
        healthcheck: input.healthcheck,
        sentinelConfig: "{}",
        createdAt: Date.now(),
        updatedAt: Date.now(),
      })
      .onDuplicateKeyUpdate({ set: { healthcheck: input.healthcheck, updatedAt: Date.now() } });
  });
