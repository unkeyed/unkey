import { and, db, eq, schema, sql } from "@/lib/db";
import { ratelimit, withRatelimit, workspaceProcedure } from "@/lib/trpc/trpc";
import { TRPCError } from "@trpc/server";
import { z } from "zod";

export const queryComputeAllocationResponse = z.object({
  totalCpuMillicores: z.number(),
  totalMemoryMib: z.number(),
  totalStorageMib: z.number(),
});

export type ComputeAllocationResponse = z.infer<typeof queryComputeAllocationResponse>;

/**
 * SUM aliases come back as MySQL BIGINT, which mysql2 hands over as a string
 * once it exceeds the safe integer range, so parse instead of asserting a type.
 */
const allocationRow = z.object({
  totalCpuMillicores: z.coerce.number(),
  totalMemoryMib: z.coerce.number(),
  totalStorageMib: z.coerce.number(),
});

/**
 * Compute the workspace currently has reserved.
 *
 * Each running topology row is multiplied by autoscaling_replicas_max, so this
 * is worst-case reserved capacity if every deployment scaled all the way out —
 * not what the pods are drawing right now.
 *
 * The arithmetic deliberately mirrors the control plane's admission check (see
 * the resource admission block in svc/ctrl/worker/deploy/deploy_handler.go,
 * around lines 549-590, and its query
 * deployment_topology_sum_allocated_resources_by_workspace_id.sql), so the
 * dashboard can never show a different number from the ceiling that actually
 * rejects a deploy.
 *
 * Worst-case is the right definition here, confirmed in review: it is what the
 * workspace has actually reserved, and it is the figure the admission check
 * rejects a deploy against.
 */
export const queryComputeAllocation = workspaceProcedure
  .use(withRatelimit(ratelimit.read))
  .output(queryComputeAllocationResponse)
  .query(async ({ ctx }) => {
    try {
      const rows = await db
        .select({
          totalCpuMillicores: sql`CAST(COALESCE(SUM(${schema.deployments.cpuMillicores} * ${schema.deploymentTopology.autoscalingReplicasMax}), 0) AS SIGNED)`,
          totalMemoryMib: sql`CAST(COALESCE(SUM(${schema.deployments.memoryMib} * ${schema.deploymentTopology.autoscalingReplicasMax}), 0) AS SIGNED)`,
          totalStorageMib: sql`CAST(COALESCE(SUM(${schema.deployments.storageMib} * ${schema.deploymentTopology.autoscalingReplicasMax}), 0) AS SIGNED)`,
        })
        .from(schema.deploymentTopology)
        .innerJoin(
          schema.deployments,
          eq(schema.deployments.id, schema.deploymentTopology.deploymentId),
        )
        .where(
          and(
            eq(schema.deploymentTopology.workspaceId, ctx.workspace.id),
            eq(schema.deploymentTopology.desiredStatus, "running"),
          ),
        );

      const row = rows.at(0);
      if (!row) {
        return { totalCpuMillicores: 0, totalMemoryMib: 0, totalStorageMib: 0 };
      }
      return allocationRow.parse(row);
    } catch (err) {
      console.error("Failed to query compute allocation", err);
      throw new TRPCError({
        code: "INTERNAL_SERVER_ERROR",
        message: "Failed to fetch compute allocation. Please try again later.",
      });
    }
  });
