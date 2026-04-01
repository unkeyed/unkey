import type Stripe from "stripe";

/**
 * Validates and parses quota metadata from a Stripe product.
 * Returns parsed quota values if valid, or indicates validation failure.
 */
export function validateAndParseQuotas(product: Stripe.Product): {
  valid: boolean;
  requestsPerMonth?: number;
  logsRetentionDays?: number;
  auditLogsRetentionDays?: number;
} {
  const requiredMetadata = [
    "quota_requests_per_month",
    "quota_logs_retention_days",
    "quota_audit_logs_retention_days",
  ];

  for (const field of requiredMetadata) {
    if (!product.metadata[field]) {
      console.error(`Missing required metadata field: ${field} for product: ${product.id}`);
      return { valid: false };
    }
  }

  const requestsPerMonth = Number.parseInt(product.metadata.quota_requests_per_month);
  const logsRetentionDays = Number.parseInt(product.metadata.quota_logs_retention_days);
  const auditLogsRetentionDays = Number.parseInt(product.metadata.quota_audit_logs_retention_days);

  if (
    Number.isNaN(requestsPerMonth) ||
    Number.isNaN(logsRetentionDays) ||
    Number.isNaN(auditLogsRetentionDays)
  ) {
    console.error(`Invalid quota metadata - parsed to NaN for product: ${product.id}`);
    return { valid: false };
  }

  return {
    valid: true,
    requestsPerMonth,
    logsRetentionDays,
    auditLogsRetentionDays,
  };
}
