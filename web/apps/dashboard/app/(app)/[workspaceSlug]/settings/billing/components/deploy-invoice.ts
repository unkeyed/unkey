import { formatDollars } from "@/lib/fmt";

export type DeployInvoiceInput = {
  /** Gross month-to-date usage in cents, priced as the spend-cap worker prices it. */
  usageCents: number | null;
  /** Usage extrapolated to period end, same pricing. */
  projectedUsageCents: number | null;
  /** The plan's recurring fee from the plan catalog. */
  planFeeCents: number | null;
  /** Usage credit actually granted for this period, read from Stripe. */
  grantedCreditCents: number | null;
  /** Stripe's own preview of the next Compute invoice total. */
  previewTotalCents: number | null;
};

export type DeployInvoice = {
  periodFeeCents: number;
  includedCreditCents: number;
  feeProrated: boolean;
  usageCents: number;
  overageCents: number;
  creditRemainingCents: number;
  nextPlanFeeCents: number;
  nextInvoiceCents: number;
  /** Null unless the projection is above what has already accrued. */
  projectedNextInvoiceCents: number | null;
  periodTotalCents: number;
};

/**
 * Reconciles what the next Compute invoice charges. The plan fee grants usage
 * credit of the same amount for the period, so only usage past the credit
 * reaches the invoice, and that invoice also carries the next period's full fee.
 *
 * A mid-cycle plan change prorates both the fee and the grant, so the granted
 * amount is read rather than assumed: it is what keeps the estimate matching the
 * invoice. Falls back to the catalog fee when there is no grant to read, so a
 * missing grant understates the credit rather than the fee.
 *
 * Returns null when a figure is still missing, since a partial reconciliation
 * reads as an undercharge.
 */
export function reconcileDeployInvoice({
  usageCents,
  projectedUsageCents,
  planFeeCents,
  grantedCreditCents,
  previewTotalCents,
}: DeployInvoiceInput): DeployInvoice | null {
  const includedCreditCents = grantedCreditCents ?? planFeeCents;
  const periodFeeCents =
    includedCreditCents !== null && includedCreditCents > 0 ? includedCreditCents : planFeeCents;

  if (
    planFeeCents === null ||
    usageCents === null ||
    includedCreditCents === null ||
    periodFeeCents === null
  ) {
    return null;
  }

  const overageCents = Math.max(0, usageCents - includedCreditCents);
  const projectedOverageCents =
    projectedUsageCents === null ? null : Math.max(0, projectedUsageCents - includedCreditCents);
  const nextInvoiceCents = previewTotalCents ?? overageCents + planFeeCents;

  return {
    periodFeeCents,
    includedCreditCents,
    feeProrated: periodFeeCents !== planFeeCents,
    usageCents,
    overageCents,
    creditRemainingCents: Math.max(0, includedCreditCents - usageCents),
    nextPlanFeeCents: planFeeCents,
    nextInvoiceCents,
    projectedNextInvoiceCents:
      projectedOverageCents !== null && projectedOverageCents > overageCents
        ? nextInvoiceCents + (projectedOverageCents - overageCents)
        : null,
    periodTotalCents: periodFeeCents + overageCents,
  };
}

/**
 * The Compute plan row's subtitle: the credit is what the fee buys, so a clean
 * period states the fee. Only a prorated period states the granted amount, since
 * that is the period where the two differ — reading the grant otherwise would
 * report $0 whenever the webhook has not written one yet.
 */
export function creditLabel(planFeeCents: number, invoice: DeployInvoice | null): string {
  return invoice?.feeProrated
    ? `${formatDollars(invoice.includedCreditCents)} usage credit, prorated`
    : `${formatDollars(planFeeCents)} usage credit`;
}
