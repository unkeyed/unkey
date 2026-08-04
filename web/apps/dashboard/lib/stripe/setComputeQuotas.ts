import { limitsByPlan } from "@/lib/quotas";
import {
  type Database,
  type InsertLimits,
  type InsertQuotas,
  type Transaction,
  schema,
} from "@unkey/db";
import { type DeployPlan, computeQuotaUpdateForPlan } from "./deployPlan";

/** Updates quota and writes the matching limits row. */
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

  const planLimits = limitsByPlan[params.plan ?? "free"];
  const quotaUpdate = params.quotaUpdate ?? {};
  const limitValues = {
    workspaceId: params.workspaceId,
    apiBillableOperationsCountMaxPerMonth:
      quotaUpdate.requestsPerMonth !== undefined
        ? quotaUpdate.requestsPerMonth
        : planLimits.apiBillableOperationsCountMaxPerMonth,
    apiRequestsCountMaxPerMinute:
      quotaUpdate.ratelimitApiLimit !== undefined
        ? quotaUpdate.ratelimitApiLimit
        : planLimits.apiRequestsCountMaxPerMinute,
    logsRetentionDaysMax:
      quotaUpdate.logsRetentionDays !== undefined
        ? quotaUpdate.logsRetentionDays
        : planLimits.logsRetentionDaysMax,
    logsAuditRetentionDaysMax:
      quotaUpdate.auditLogsRetentionDays !== undefined
        ? quotaUpdate.auditLogsRetentionDays
        : planLimits.logsAuditRetentionDaysMax,
    teamEnabled: quotaUpdate.team !== undefined ? quotaUpdate.team : planLimits.teamEnabled,
    cpuCoresMax:
      quotaUpdate.allocatedCpuMillicoresTotal !== undefined
        ? Math.ceil(quotaUpdate.allocatedCpuMillicoresTotal / 1_000)
        : planLimits.cpuCoresMax,
    cpuCoresMaxPerInstance:
      quotaUpdate.maxCpuMillicoresPerInstance !== undefined
        ? Math.ceil(quotaUpdate.maxCpuMillicoresPerInstance / 1_000)
        : planLimits.cpuCoresMaxPerInstance,
    memoryMibMax:
      quotaUpdate.allocatedMemoryMibTotal !== undefined
        ? quotaUpdate.allocatedMemoryMibTotal
        : planLimits.memoryMibMax,
    memoryMibMaxPerInstance:
      quotaUpdate.maxMemoryMibPerInstance !== undefined
        ? quotaUpdate.maxMemoryMibPerInstance
        : planLimits.memoryMibMaxPerInstance,
    storageMibMax:
      quotaUpdate.allocatedStorageMibTotal !== undefined
        ? quotaUpdate.allocatedStorageMibTotal
        : planLimits.storageMibMax,
    storageMibMaxPerInstance:
      quotaUpdate.maxStorageMibPerInstance !== undefined
        ? quotaUpdate.maxStorageMibPerInstance
        : planLimits.storageMibMaxPerInstance,
    buildsConcurrentMax:
      quotaUpdate.maxConcurrentBuilds !== undefined
        ? quotaUpdate.maxConcurrentBuilds
        : planLimits.buildsConcurrentMax,
    customDomainsMax: planLimits.customDomainsMax,
    autoscalingReplicasMax: planLimits.autoscalingReplicasMax,
  };
  const limitUpdate: Partial<Omit<InsertLimits, "workspaceId" | "pk">> = {
    cpuCoresMax: limitValues.cpuCoresMax,
    cpuCoresMaxPerInstance: limitValues.cpuCoresMaxPerInstance,
    memoryMibMax: limitValues.memoryMibMax,
    memoryMibMaxPerInstance: limitValues.memoryMibMaxPerInstance,
    storageMibMax: limitValues.storageMibMax,
    storageMibMaxPerInstance: limitValues.storageMibMaxPerInstance,
    buildsConcurrentMax: limitValues.buildsConcurrentMax,
    customDomainsMax: limitValues.customDomainsMax,
    autoscalingReplicasMax: limitValues.autoscalingReplicasMax,
  };

  if (!params.preserveApiQuotas || quotaUpdate.requestsPerMonth !== undefined) {
    limitUpdate.apiBillableOperationsCountMaxPerMonth =
      limitValues.apiBillableOperationsCountMaxPerMonth;
  }
  if (!params.preserveApiQuotas || quotaUpdate.ratelimitApiLimit !== undefined) {
    limitUpdate.apiRequestsCountMaxPerMinute = limitValues.apiRequestsCountMaxPerMinute;
  }
  if (!params.preserveApiQuotas || quotaUpdate.logsRetentionDays !== undefined) {
    limitUpdate.logsRetentionDaysMax = limitValues.logsRetentionDaysMax;
  }
  if (!params.preserveApiQuotas || quotaUpdate.auditLogsRetentionDays !== undefined) {
    limitUpdate.logsAuditRetentionDaysMax = limitValues.logsAuditRetentionDaysMax;
  }
  if (!params.preserveApiQuotas || quotaUpdate.team !== undefined) {
    limitUpdate.teamEnabled = limitValues.teamEnabled;
  }

  await db.insert(schema.limits).values(limitValues).onDuplicateKeyUpdate({ set: limitUpdate });
}
