import { type Database, type Transaction, schema } from "@unkey/db";
import { type DeployPlan, computeQuotaUpdateForPlan } from "./deployPlan";

/** Applies Compute plan quotas without weakening a paid API plan's shared entitlements. */
export async function setComputeQuotas(
  db: Transaction | Database,
  params: { workspaceId: string; plan: DeployPlan | null; preserveApiQuotas: boolean },
): Promise<void> {
  const quotas = computeQuotaUpdateForPlan(params.plan, params.preserveApiQuotas);
  await db
    .insert(schema.quotas)
    .values({ workspaceId: params.workspaceId, ...quotas })
    .onDuplicateKeyUpdate({ set: quotas });
}
