export function periodCredit(
  planFeeCents: number | null,
  grantedCreditCents: number | null,
): { cents: number; prorated: boolean } | null {
  const cents = grantedCreditCents ?? planFeeCents;
  if (planFeeCents === null || cents === null) {
    return null;
  }
  const resolved = cents > 0 ? cents : planFeeCents;
  return { cents: resolved, prorated: resolved !== planFeeCents };
}
