import { limitsByPlan } from "@/lib/limits";
import { type Database, type InsertLimits, type Transaction, schema } from "@unkey/db";
import { type DeployPlan, computeLimitUpdateForPlan } from "./deployPlan";

type LimitUpdate = Partial<Omit<InsertLimits, "workspaceId" | "pk">>;

/** Updates the limits row for API and Compute plan changes. */
export async function setWorkspaceLimits(
  db: Transaction | Database,
  params: {
    workspaceId: string;
    plan: DeployPlan | null;
    preserveApiLimits: boolean;
    limitUpdate?: LimitUpdate;
  },
): Promise<void> {
  const requestedLimits = {
    ...computeLimitUpdateForPlan(params.plan, params.preserveApiLimits),
    ...params.limitUpdate,
  };
  const planLimits = limitsByPlan[params.plan ?? "free"];
  const hasRequestedLimit = (key: keyof LimitUpdate) =>
    Object.prototype.hasOwnProperty.call(requestedLimits, key);
  const fromUpdate = <K extends keyof LimitUpdate>(key: K): LimitUpdate[K] =>
    hasRequestedLimit(key) ? requestedLimits[key] : planLimits[key];

  const limitValues = {
    workspaceId: params.workspaceId,
    apiBillableOperationsCountMaxPerMonth:
      fromUpdate("apiBillableOperationsCountMaxPerMonth") ??
      planLimits.apiBillableOperationsCountMaxPerMonth,
    apiRequestsCountMaxPerMinute: fromUpdate("apiRequestsCountMaxPerMinute") ?? null,
    logsRetentionDaysMax: fromUpdate("logsRetentionDaysMax") ?? planLimits.logsRetentionDaysMax,
    logsAuditRetentionDaysMax:
      fromUpdate("logsAuditRetentionDaysMax") ?? planLimits.logsAuditRetentionDaysMax,
    teamEnabled: fromUpdate("teamEnabled") ?? planLimits.teamEnabled,
    cpuCoresMax: fromUpdate("cpuCoresMax") ?? planLimits.cpuCoresMax,
    cpuCoresMaxPerInstance:
      fromUpdate("cpuCoresMaxPerInstance") ?? planLimits.cpuCoresMaxPerInstance,
    memoryMibMax: fromUpdate("memoryMibMax") ?? planLimits.memoryMibMax,
    memoryMibMaxPerInstance:
      fromUpdate("memoryMibMaxPerInstance") ?? planLimits.memoryMibMaxPerInstance,
    storageMibMax: fromUpdate("storageMibMax") ?? planLimits.storageMibMax,
    storageMibMaxPerInstance:
      fromUpdate("storageMibMaxPerInstance") ?? planLimits.storageMibMaxPerInstance,
    buildsConcurrentMax: fromUpdate("buildsConcurrentMax") ?? planLimits.buildsConcurrentMax,
    customDomainsMax: fromUpdate("customDomainsMax") ?? planLimits.customDomainsMax,
    autoscalingReplicasMax:
      fromUpdate("autoscalingReplicasMax") ?? planLimits.autoscalingReplicasMax,
  };
  const limitUpdate: LimitUpdate = {
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

  if (!params.preserveApiLimits || hasRequestedLimit("apiBillableOperationsCountMaxPerMonth")) {
    limitUpdate.apiBillableOperationsCountMaxPerMonth =
      limitValues.apiBillableOperationsCountMaxPerMonth;
  }
  if (!params.preserveApiLimits || hasRequestedLimit("apiRequestsCountMaxPerMinute")) {
    limitUpdate.apiRequestsCountMaxPerMinute = limitValues.apiRequestsCountMaxPerMinute;
  }
  if (!params.preserveApiLimits || hasRequestedLimit("logsRetentionDaysMax")) {
    limitUpdate.logsRetentionDaysMax = limitValues.logsRetentionDaysMax;
  }
  if (!params.preserveApiLimits || hasRequestedLimit("logsAuditRetentionDaysMax")) {
    limitUpdate.logsAuditRetentionDaysMax = limitValues.logsAuditRetentionDaysMax;
  }
  if (!params.preserveApiLimits || hasRequestedLimit("teamEnabled")) {
    limitUpdate.teamEnabled = limitValues.teamEnabled;
  }

  await db.insert(schema.limits).values(limitValues).onDuplicateKeyUpdate({ set: limitUpdate });
}
