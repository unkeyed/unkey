import { type Database, type Transaction, schema } from "@unkey/db";
import { type DeployPlan, computeQuotasForPlan } from "./deployPlan";

/** Applies only Compute-owned quota fields, preserving any active API plan's quotas. */
export async function setComputeQuotas(
  db: Transaction | Database,
  params: { workspaceId: string; plan: DeployPlan | null },
): Promise<void> {
  const quotas = computeQuotasForPlan(params.plan);
  await db
    .insert(schema.quotas)
    .values({ workspaceId: params.workspaceId, ...quotas })
    .onDuplicateKeyUpdate({ set: quotas });
}
