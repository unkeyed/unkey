import type { InsertLimits, Quotas } from "@unkey/db";
import type { DeployPlan } from "./stripe/deployPlan";

export type PlanLimits = Omit<InsertLimits, "workspaceId" | "pk">;
export type LimitsPlan = "free" | DeployPlan;

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
    diskEphemeralMibMax: 51_200,
    diskEphemeralMibMaxPerInstance: 10_240,
    buildsConcurrentCountMax: 1,
    customDomainsCountMax: 0,
  },
  starter: {
    apiBillableOperationsCountMaxPerMonth: 150_000,
    apiRequestsCountMaxPerMinute: null,
    logsRetentionDaysMax: 3,
    logsAuditRetentionDaysMax: 7,
    teamEnabled: false,
    cpuCoresMax: 10,
    cpuCoresMaxPerInstance: 2,
    memoryMibMax: 20_480,
    memoryMibMaxPerInstance: 2_048,
    diskEphemeralMibMax: 51_200,
    diskEphemeralMibMaxPerInstance: 10_240,
    buildsConcurrentCountMax: 1,
    customDomainsCountMax: 1,
  },
  pro: {
    apiBillableOperationsCountMaxPerMonth: 150_000,
    apiRequestsCountMaxPerMinute: null,
    logsRetentionDaysMax: 7,
    logsAuditRetentionDaysMax: 14,
    teamEnabled: true,
    cpuCoresMax: 10,
    cpuCoresMaxPerInstance: 8,
    memoryMibMax: 20_480,
    memoryMibMaxPerInstance: 8_192,
    diskEphemeralMibMax: 51_200,
    diskEphemeralMibMaxPerInstance: 10_240,
    buildsConcurrentCountMax: 1,
    customDomainsCountMax: 1_000_000,
  },
  business: {
    apiBillableOperationsCountMaxPerMonth: 150_000,
    apiRequestsCountMaxPerMinute: null,
    logsRetentionDaysMax: 14,
    logsAuditRetentionDaysMax: 30,
    teamEnabled: true,
    cpuCoresMax: 10,
    cpuCoresMaxPerInstance: 16,
    memoryMibMax: 20_480,
    memoryMibMaxPerInstance: 32_768,
    diskEphemeralMibMax: 51_200,
    diskEphemeralMibMaxPerInstance: 10_240,
    buildsConcurrentCountMax: 1,
    customDomainsCountMax: 1_000_000,
  },
} satisfies Record<LimitsPlan, PlanLimits>;

export const freeTierLimits: PlanLimits = limitsByPlan.free;

export const freeTierQuotas: Omit<Quotas, "workspaceId" | "pk"> = {
  requestsPerMonth: limitsByPlan.free.apiBillableOperationsCountMaxPerMonth,
  logsRetentionDays: limitsByPlan.free.logsRetentionDaysMax,
  auditLogsRetentionDays: limitsByPlan.free.logsAuditRetentionDaysMax,
  team: limitsByPlan.free.teamEnabled,
  ratelimitApiDuration: null,
  ratelimitApiLimit: limitsByPlan.free.apiRequestsCountMaxPerMinute,
  allocatedCpuMillicoresTotal: limitsByPlan.free.cpuCoresMax * 1_000,
  allocatedMemoryMibTotal: limitsByPlan.free.memoryMibMax,
  allocatedStorageMibTotal: limitsByPlan.free.diskEphemeralMibMax,
  maxCpuMillicoresPerInstance: limitsByPlan.free.cpuCoresMaxPerInstance * 1_000,
  maxMemoryMibPerInstance: limitsByPlan.free.memoryMibMaxPerInstance,
  maxStorageMibPerInstance: limitsByPlan.free.diskEphemeralMibMaxPerInstance,
  maxConcurrentBuilds: limitsByPlan.free.buildsConcurrentCountMax,
  maxReplicasPerRegion: 4,
};
