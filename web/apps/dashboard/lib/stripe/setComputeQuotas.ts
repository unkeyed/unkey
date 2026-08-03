import { type Database, type InsertQuotas, type Transaction, schema } from "@unkey/db";
import { type DeployPlan, computeQuotaUpdateForPlan } from "./deployPlan";

const customDomainLimitByPlan = {
  starter: 1,
  pro: 1_000_000,
  business: 1_000_000,
} satisfies Record<DeployPlan, number>;

/** Updates quota once, then mirrors the resulting workspace limits into limits. */
export async function setComputeQuotas(
  db: Transaction | Database,
  params: {
    workspaceId: string;
    plan: DeployPlan | null;
    preserveApiQuotas: boolean;
    quotaUpdate?: Partial<Omit<InsertQuotas, "workspaceId" | "pk">>;
  },
): Promise<void> {
  const quotas = {
    ...computeQuotaUpdateForPlan(params.plan, params.preserveApiQuotas),
    ...params.quotaUpdate,
  };
  await db
    .insert(schema.quotas)
    .values({ workspaceId: params.workspaceId, ...quotas })
    .onDuplicateKeyUpdate({ set: quotas });

  const quota = await db.query.quotas.findFirst({
    where: (table, { eq }) => eq(table.workspaceId, params.workspaceId),
  });
  if (!quota) {
    throw new Error(`Quota row missing after quota update for ${params.workspaceId}`);
  }

  const limitValues = {
    workspaceId: params.workspaceId,
    apiBillableOperationsCountMaxPerMonth: quota.requestsPerMonth,
    apiRequestsCountMaxPerMinute: quota.ratelimitApiLimit,
    logsRetentionDaysMax: quota.logsRetentionDays,
    logsAuditRetentionDaysMax: quota.auditLogsRetentionDays,
    teamEnabled: quota.team,
    cpuCoresMax: Math.ceil(quota.allocatedCpuMillicoresTotal / 1_000),
    cpuCoresMaxPerInstance: Math.ceil(quota.maxCpuMillicoresPerInstance / 1_000),
    memoryMibMax: quota.allocatedMemoryMibTotal,
    memoryMibMaxPerInstance: quota.maxMemoryMibPerInstance,
    diskEphemeralMibMax: quota.allocatedStorageMibTotal,
    diskEphemeralMibMaxPerInstance: quota.maxStorageMibPerInstance,
    buildsConcurrentCountMax: quota.maxConcurrentBuilds,
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
        cpuCoresMax: limitValues.cpuCoresMax,
        cpuCoresMaxPerInstance: limitValues.cpuCoresMaxPerInstance,
        memoryMibMax: limitValues.memoryMibMax,
        memoryMibMaxPerInstance: limitValues.memoryMibMaxPerInstance,
        diskEphemeralMibMax: limitValues.diskEphemeralMibMax,
        diskEphemeralMibMaxPerInstance: limitValues.diskEphemeralMibMaxPerInstance,
        buildsConcurrentCountMax: limitValues.buildsConcurrentCountMax,
        customDomainsCountMax: limitValues.customDomainsCountMax,
      },
    });
}
