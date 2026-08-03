import { type Database, type Transaction, schema } from "@unkey/db";
import { type DeployPlan, computeQuotaUpdateForPlan } from "./deployPlan";

const customDomainLimitByPlan = {
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
    cpuMax: Math.ceil(legacy.allocatedCpuMillicoresTotal / 1_000),
    cpuMaxPerInstance: Math.ceil(legacy.maxCpuMillicoresPerInstance / 1_000),
    memoryMibMax: legacy.allocatedMemoryMibTotal,
    memoryMibMaxPerInstance: legacy.maxMemoryMibPerInstance,
    diskEphemeralMibMax: legacy.allocatedStorageMibTotal,
    diskEphemeralMibMaxPerInstance: legacy.maxStorageMibPerInstance,
    buildsConcurrentCountMax: legacy.maxConcurrentBuilds,
    customDomainsCountMax: params.plan ? customDomainLimitByPlan[params.plan] : 0,
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
        cpuMax: limitValues.cpuMax,
        cpuMaxPerInstance: limitValues.cpuMaxPerInstance,
        memoryMibMax: limitValues.memoryMibMax,
        memoryMibMaxPerInstance: limitValues.memoryMibMaxPerInstance,
        diskEphemeralMibMax: limitValues.diskEphemeralMibMax,
        diskEphemeralMibMaxPerInstance: limitValues.diskEphemeralMibMaxPerInstance,
        buildsConcurrentCountMax: limitValues.buildsConcurrentCountMax,
        customDomainsCountMax: limitValues.customDomainsCountMax,
      },
    });
}
