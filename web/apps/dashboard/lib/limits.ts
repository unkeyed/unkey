import type { Limits } from "@unkey/db";
import type { DeployPlan } from "./stripe/deployPlan";

export type PlanLimits = Omit<Limits, "workspaceId" | "pk">;
export type LimitsPlan = "free" | DeployPlan;

/**
 * The cap the plans below use when custom domains are effectively uncapped. It
 * exists to stop abuse, not to price the feature, so the pricing page and the
 * dashboard both call this "Unlimited". A real number keeps the gate in ctrl
 * simple: it compares a count against one column.
 */
export const CUSTOM_DOMAINS_UNLIMITED = 1_000_000;

export const limitsByPlan = {
  free: {
    apiBillableOperationsCountMaxPerMonth: 150_000,
    apiRequestsCountMaxPerMinute: null,
    logsRetentionDaysMax: 7,
    logsAuditRetentionDaysMax: 30,
    teamEnabled: false,
    cpuCoresMax: 10,
    cpuCoresMaxPerInstance: 2,
    memoryMibMax: 20_480,
    memoryMibMaxPerInstance: 4_096,
    storageMibMax: 51_200,
    storageMibMaxPerInstance: 10_240,
    buildsConcurrentMax: 1,
    customDomainsMax: 0,
    autoscalingReplicasMax: 0,
  },
  starter: {
    apiBillableOperationsCountMaxPerMonth: 150_000,
    apiRequestsCountMaxPerMinute: null,
    logsRetentionDaysMax: 3,
    logsAuditRetentionDaysMax: 7,
    teamEnabled: false,
    cpuCoresMax: 30,
    cpuCoresMaxPerInstance: 2,
    memoryMibMax: 61_440,
    memoryMibMaxPerInstance: 2_048,
    storageMibMax: 122_880,
    storageMibMaxPerInstance: 10_240,
    buildsConcurrentMax: 1,
    customDomainsMax: 1,
    autoscalingReplicasMax: 4,
  },
  pro: {
    apiBillableOperationsCountMaxPerMonth: 150_000,
    apiRequestsCountMaxPerMinute: null,
    logsRetentionDaysMax: 7,
    logsAuditRetentionDaysMax: 14,
    teamEnabled: true,
    cpuCoresMax: 120,
    cpuCoresMaxPerInstance: 8,
    memoryMibMax: 245_760,
    memoryMibMaxPerInstance: 8_192,
    storageMibMax: 491_520,
    storageMibMaxPerInstance: 10_240,
    buildsConcurrentMax: 1,
    customDomainsMax: CUSTOM_DOMAINS_UNLIMITED,
    autoscalingReplicasMax: 8,
  },
  business: {
    apiBillableOperationsCountMaxPerMonth: 150_000,
    apiRequestsCountMaxPerMinute: null,
    logsRetentionDaysMax: 14,
    logsAuditRetentionDaysMax: 30,
    teamEnabled: true,
    cpuCoresMax: 240,
    cpuCoresMaxPerInstance: 16,
    memoryMibMax: 491_520,
    memoryMibMaxPerInstance: 32_768,
    storageMibMax: 983_040,
    storageMibMaxPerInstance: 10_240,
    buildsConcurrentMax: 1,
    customDomainsMax: CUSTOM_DOMAINS_UNLIMITED,
    autoscalingReplicasMax: 16,
  },
} satisfies Record<LimitsPlan, PlanLimits>;

export const freeTierLimits: PlanLimits = limitsByPlan.free;
