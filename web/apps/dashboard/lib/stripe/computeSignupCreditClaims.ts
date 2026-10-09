import { and, db, eq, isDuplicateKeyError, isNull, or, primaryDb } from "@/lib/db";
import { computeSignupCreditClaims as claims } from "@unkey/db/src/schema";

export type ComputeSignupCreditClaim = typeof claims.$inferSelect;

export type ClaimKey = {
  attemptId: string;
  workspaceId: string;
  cardFingerprint: string;
  workosUserId: string;
};

export type ClaimIdentity = Omit<ClaimKey, "attemptId">;

export type ClaimConflicts = {
  byWorkspace: ComputeSignupCreditClaim | null;
  byFingerprint: ComputeSignupCreditClaim | null;
  byUser: ComputeSignupCreditClaim | null;
};

function matchesKey(claim: ClaimKey) {
  return and(
    eq(claims.attemptId, claim.attemptId),
    eq(claims.workspaceId, claim.workspaceId),
    eq(claims.cardFingerprint, claim.cardFingerprint),
    eq(claims.workosUserId, claim.workosUserId),
    isNull(claims.stripeBalanceTransactionId),
  );
}

// Claim reads feed compare-and-swap on attempt_id right after writes, so they must not hit a lagging replica.
export async function findClaimByWorkspace(
  workspaceId: string,
): Promise<ComputeSignupCreditClaim | null> {
  const rows = await primaryDb
    .select()
    .from(claims)
    .where(eq(claims.workspaceId, workspaceId))
    .limit(1);
  return rows[0] ?? null;
}

export async function findClaimConflicts(identity: ClaimIdentity): Promise<ClaimConflicts> {
  const rows = await primaryDb
    .select()
    .from(claims)
    .where(
      or(
        eq(claims.workspaceId, identity.workspaceId),
        eq(claims.cardFingerprint, identity.cardFingerprint),
        eq(claims.workosUserId, identity.workosUserId),
      ),
    )
    .limit(3);
  return {
    byWorkspace: rows.find((row) => row.workspaceId === identity.workspaceId) ?? null,
    byFingerprint: rows.find((row) => row.cardFingerprint === identity.cardFingerprint) ?? null,
    byUser: rows.find((row) => row.workosUserId === identity.workosUserId) ?? null,
  };
}

export async function hasClaimForWorkspaceOrUser(
  workspaceId: string,
  workosUserId: string,
): Promise<boolean> {
  const rows = await db
    .select({ pk: claims.pk })
    .from(claims)
    .where(or(eq(claims.workspaceId, workspaceId), eq(claims.workosUserId, workosUserId)))
    .limit(1);
  return rows.length > 0;
}

export async function insertClaim(
  row: Omit<typeof claims.$inferInsert, "pk">,
): Promise<"inserted" | "duplicate"> {
  try {
    await db.insert(claims).values(row);
    return "inserted";
  } catch (error) {
    if (isDuplicateKeyError(error)) {
      return "duplicate";
    }
    throw error;
  }
}

export async function takeOverClaim(
  seen: Pick<ComputeSignupCreditClaim, "pk" | "attemptId">,
  next: ClaimKey & { stripeCustomerId: string; createdAt: number },
): Promise<boolean> {
  try {
    const result = await db
      .update(claims)
      .set(next)
      .where(
        and(
          eq(claims.pk, seen.pk),
          eq(claims.attemptId, seen.attemptId),
          isNull(claims.stripeBalanceTransactionId),
        ),
      );
    return result[0].affectedRows === 1;
  } catch (error) {
    if (isDuplicateKeyError(error)) {
      return false;
    }
    throw error;
  }
}

export async function setClaimTransactionId(
  claim: ClaimKey,
  transactionId: string,
): Promise<boolean> {
  const result = await db
    .update(claims)
    .set({ stripeBalanceTransactionId: transactionId })
    .where(matchesKey(claim));
  return result[0].affectedRows === 1;
}

export async function deleteClaimAttempt(claim: ClaimKey): Promise<void> {
  await db.delete(claims).where(matchesKey(claim));
}
