import type { Quotas } from "@unkey/db";

export const freeTierQuotas: Omit<Quotas, "workspaceId"> = {
  requestsPerMonth: 150_000,
  logsRetentionDays: 7,
  auditLogsRetentionDays: 30,
  team: false,
  applySubscriptionChanges: true,
};
