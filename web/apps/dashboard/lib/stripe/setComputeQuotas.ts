import { type Database, type Transaction, schema } from "@unkey/db";
import { type DeployPlan, computeQuotaUpdateForPlan } from "./deployPlan";

const computeCustomDomainLimitByPlan = {
  starter: 1,
  pro: 1_000_000,
  business: 1_000_000,
} satisfies Record<DeployPlan, number>;

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

  const legacy = await db.query.quotas.findFirst({
    where: (table, { eq }) => eq(table.workspaceId, params.workspaceId),
  });
  if (!legacy) {
    throw new Error(
      `Legacy quota row missing after Compute quota update for ${params.workspaceId}`,
    );
  }

  const limitValues = {
    workspaceId: params.workspaceId,
    apiBillableOperationsCountMaxPerMonth: legacy.requestsPerMonth,
    apiRequestsCountMaxPerMinute: legacy.ratelimitApiLimit,
    logsRetentionDaysMax: legacy.logsRetentionDays,
    logsAuditRetentionDaysMax: legacy.auditLogsRetentionDays,
    teamEnabled: legacy.team,
    computeCpuMax: Math.ceil(legacy.allocatedCpuMillicoresTotal / 1_000),
    computeCpuMaxPerInstance: Math.ceil(legacy.maxCpuMillicoresPerInstance / 1_000),
    computeMemoryMibMax: legacy.allocatedMemoryMibTotal,
    computeMemoryMibMaxPerInstance: legacy.maxMemoryMibPerInstance,
    computeDiskEphemeralMibMax: legacy.allocatedStorageMibTotal,
    computeDiskEphemeralMibMaxPerInstance: legacy.maxStorageMibPerInstance,
    computeBuildsConcurrentCountMax: legacy.maxConcurrentBuilds,
    computeCustomDomainsCountMax: params.plan ? computeCustomDomainLimitByPlan[params.plan] : 0,
  };
  await db
    .insert(schema.limits)
    .values(limitValues)
    .onDuplicateKeyUpdate({
      set: {
        apiBillableOperationsCountMaxPerMonth: limitValues.apiBillableOperationsCountMaxPerMonth,
        apiRequestsCountMaxPerMinute: limitValues.apiRequestsCountMaxPerMinute,
        logsRetentionDaysMax: limitValues.logsRetentionDaysMax,
        logsAuditRetentionDaysMax: limitValues.logsAuditRetentionDaysMax,
        teamEnabled: limitValues.teamEnabled,
        computeCpuMax: limitValues.computeCpuMax,
        computeCpuMaxPerInstance: limitValues.computeCpuMaxPerInstance,
        computeMemoryMibMax: limitValues.computeMemoryMibMax,
        computeMemoryMibMaxPerInstance: limitValues.computeMemoryMibMaxPerInstance,
        computeDiskEphemeralMibMax: limitValues.computeDiskEphemeralMibMax,
        computeDiskEphemeralMibMaxPerInstance: limitValues.computeDiskEphemeralMibMaxPerInstance,
        computeBuildsConcurrentCountMax: limitValues.computeBuildsConcurrentCountMax,
        computeCustomDomainsCountMax: limitValues.computeCustomDomainsCountMax,
      },
    });
}
