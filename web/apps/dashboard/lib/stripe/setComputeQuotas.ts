import { limitsByPlan } from "@/lib/quotas";
import { type Database, type InsertQuotas, type Transaction, schema } from "@unkey/db";
import { type DeployPlan, computeQuotaUpdateForPlan } from "./deployPlan";

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

  const planLimits = limitsByPlan[params.plan ?? "free"];
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
    storageMibMax: quota.allocatedStorageMibTotal,
    storageMibMaxPerInstance: quota.maxStorageMibPerInstance,
    buildsConcurrentMax: quota.maxConcurrentBuilds,
    customDomainsMax: planLimits.customDomainsMax,
    autoscalingReplicasMax: planLimits.autoscalingReplicasMax,
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
        storageMibMax: limitValues.storageMibMax,
        storageMibMaxPerInstance: limitValues.storageMibMaxPerInstance,
        buildsConcurrentMax: limitValues.buildsConcurrentMax,
        customDomainsMax: limitValues.customDomainsMax,
        autoscalingReplicasMax: limitValues.autoscalingReplicasMax,
      },
    });
}
