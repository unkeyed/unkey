import type { InsertLimits, Quotas } from "@unkey/db";

export const freeTierQuotas: Omit<Quotas, "workspaceId" | "pk"> = {
  requestsPerMonth: 150_000,
  logsRetentionDays: 7,
  auditLogsRetentionDays: 30,
  team: false,
  ratelimitApiDuration: null,
  ratelimitApiLimit: null,
  allocatedCpuMillicoresTotal: 10000, // 10 cores
  allocatedMemoryMibTotal: 20480, // 20 GiB
  allocatedStorageMibTotal: 51200, // 50 GiB
  maxCpuMillicoresPerInstance: 2000, // 2 vCPU
  maxMemoryMibPerInstance: 4096, // 4 GiB
  maxStorageMibPerInstance: 10240, // 10 GiB
  maxConcurrentBuilds: 1,
  maxReplicasPerRegion: 4,
};

export const freeTierLimits: Omit<InsertLimits, "workspaceId" | "pk"> = {
  apiBillableOperationsCountMaxPerMonth: freeTierQuotas.requestsPerMonth,
  apiRequestsCountMaxPerMinute: freeTierQuotas.ratelimitApiLimit,
  logsRetentionDaysMax: freeTierQuotas.logsRetentionDays,
  logsAuditRetentionDaysMax: freeTierQuotas.auditLogsRetentionDays,
  teamEnabled: freeTierQuotas.team,
  cpuCoresMax: Math.ceil(freeTierQuotas.allocatedCpuMillicoresTotal / 1_000),
  cpuCoresMaxPerInstance: Math.ceil(freeTierQuotas.maxCpuMillicoresPerInstance / 1_000),
  memoryMibMax: freeTierQuotas.allocatedMemoryMibTotal,
  memoryMibMaxPerInstance: freeTierQuotas.maxMemoryMibPerInstance,
  diskEphemeralMibMax: freeTierQuotas.allocatedStorageMibTotal,
  diskEphemeralMibMaxPerInstance: freeTierQuotas.maxStorageMibPerInstance,
  buildsConcurrentCountMax: freeTierQuotas.maxConcurrentBuilds,
  customDomainsCountMax: 0,
};
