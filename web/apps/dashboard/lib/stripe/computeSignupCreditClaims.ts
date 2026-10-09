import { and, db, eq, isDuplicateKeyError, isNull, or, schema } from "@/lib/db";

export type ComputeSignupCreditClaim = {
  pk: number;
  cardFingerprint: string;
  workspaceId: string;
  stripeCustomerId: string;
  stripeBalanceTransactionId: string | null;
  workosUserId: string;
  attemptId: string;
  createdAt: number;
};

export type ClaimAttempt = {
  pk: number;
  seenAttemptId: string;
  attemptId: string;
  createdAt: number;
  workspaceId: string;
  cardFingerprint: string;
  workosUserId: string;
  stripeCustomerId: string;
};

export type ComputeSignupCreditClaimStore = {
  findByWorkspace(workspaceId: string): Promise<ComputeSignupCreditClaim | null>;
  findByFingerprint(fingerprint: string): Promise<ComputeSignupCreditClaim | null>;
  findByUser(workosUserId: string): Promise<ComputeSignupCreditClaim | null>;
  existsForWorkspaceOrUser(workspaceId: string, workosUserId: string): Promise<boolean>;
  insert(row: {
    cardFingerprint: string;
    workspaceId: string;
    stripeCustomerId: string;
    workosUserId: string;
    createdAt: number;
    attemptId: string;
  }): Promise<"inserted" | "duplicate">;
  claimAttempt(attempt: ClaimAttempt): Promise<boolean>;
  setTransactionId(input: {
    workspaceId: string;
    cardFingerprint: string;
    workosUserId: string;
    attemptId: string;
    transactionId: string;
  }): Promise<number>;
  deleteAttempt(claim: {
    attemptId: string;
    workspaceId: string;
    cardFingerprint: string;
    workosUserId: string;
  }): Promise<void>;
};

function primary(): typeof db {
  if ("$primary" in db) {
    return db.$primary as typeof db;
  }
  return db;
}

function toClaim(
  row: typeof schema.computeSignupCreditClaims.$inferSelect,
): ComputeSignupCreditClaim {
  return {
    pk: row.pk,
    cardFingerprint: row.cardFingerprint,
    workspaceId: row.workspaceId,
    stripeCustomerId: row.stripeCustomerId,
    stripeBalanceTransactionId: row.stripeBalanceTransactionId,
    workosUserId: row.workosUserId,
    attemptId: row.attemptId,
    createdAt: row.createdAt,
  };
}

function affectedRows(result: unknown): number {
  if (
    Array.isArray(result) &&
    result[0] &&
    typeof result[0] === "object" &&
    "affectedRows" in result[0]
  ) {
    const rows = result[0].affectedRows;
    return typeof rows === "number" ? rows : 0;
  }
  if (result && typeof result === "object" && "affectedRows" in result) {
    const rows = result.affectedRows;
    return typeof rows === "number" ? rows : 0;
  }
  return 0;
}

export const drizzleComputeSignupCreditClaimStore: ComputeSignupCreditClaimStore = {
  async findByWorkspace(workspaceId) {
    const claim = schema.computeSignupCreditClaims;
    const rows = await primary()
      .select({
        pk: claim.pk,
        cardFingerprint: claim.cardFingerprint,
        workspaceId: claim.workspaceId,
        stripeCustomerId: claim.stripeCustomerId,
        stripeBalanceTransactionId: claim.stripeBalanceTransactionId,
        workosUserId: claim.workosUserId,
        attemptId: claim.attemptId,
        createdAt: claim.createdAt,
      })
      .from(claim)
      .where(eq(claim.workspaceId, workspaceId))
      .limit(1);
    const row = rows[0];
    return row ? toClaim(row) : null;
  },

  // One read. workspace_id and workos_user_id are each unique, so either match is the claim.
  async existsForWorkspaceOrUser(workspaceId, workosUserId) {
    const claim = schema.computeSignupCreditClaims;
    const rows = await primary()
      .select({ pk: claim.pk })
      .from(claim)
      .where(or(eq(claim.workspaceId, workspaceId), eq(claim.workosUserId, workosUserId)))
      .limit(1);
    return rows.length > 0;
  },

  async findByFingerprint(fingerprint) {
    const claim = schema.computeSignupCreditClaims;
    const rows = await primary()
      .select({
        pk: claim.pk,
        cardFingerprint: claim.cardFingerprint,
        workspaceId: claim.workspaceId,
        stripeCustomerId: claim.stripeCustomerId,
        stripeBalanceTransactionId: claim.stripeBalanceTransactionId,
        workosUserId: claim.workosUserId,
        attemptId: claim.attemptId,
        createdAt: claim.createdAt,
      })
      .from(claim)
      .where(eq(claim.cardFingerprint, fingerprint))
      .limit(1);
    const row = rows[0];
    return row ? toClaim(row) : null;
  },

  async findByUser(workosUserId) {
    const claim = schema.computeSignupCreditClaims;
    const rows = await primary()
      .select({
        pk: claim.pk,
        cardFingerprint: claim.cardFingerprint,
        workspaceId: claim.workspaceId,
        stripeCustomerId: claim.stripeCustomerId,
        stripeBalanceTransactionId: claim.stripeBalanceTransactionId,
        workosUserId: claim.workosUserId,
        attemptId: claim.attemptId,
        createdAt: claim.createdAt,
      })
      .from(claim)
      .where(eq(claim.workosUserId, workosUserId))
      .limit(1);
    const row = rows[0];
    return row ? toClaim(row) : null;
  },

  async insert(row) {
    try {
      await primary().insert(schema.computeSignupCreditClaims).values({
        cardFingerprint: row.cardFingerprint,
        workspaceId: row.workspaceId,
        stripeCustomerId: row.stripeCustomerId,
        workosUserId: row.workosUserId,
        createdAt: row.createdAt,
        attemptId: row.attemptId,
      });
      return "inserted";
    } catch (error) {
      if (isDuplicateKeyError(error)) {
        return "duplicate";
      }
      throw error;
    }
  },

  async claimAttempt(attempt) {
    try {
      const result = await primary()
        .update(schema.computeSignupCreditClaims)
        .set({
          attemptId: attempt.attemptId,
          createdAt: attempt.createdAt,
          workspaceId: attempt.workspaceId,
          cardFingerprint: attempt.cardFingerprint,
          workosUserId: attempt.workosUserId,
          stripeCustomerId: attempt.stripeCustomerId,
        })
        .where(
          and(
            eq(schema.computeSignupCreditClaims.pk, attempt.pk),
            eq(schema.computeSignupCreditClaims.attemptId, attempt.seenAttemptId),
            isNull(schema.computeSignupCreditClaims.stripeBalanceTransactionId),
          ),
        );
      return affectedRows(result) === 1;
    } catch (error) {
      if (isDuplicateKeyError(error)) {
        return false;
      }
      throw error;
    }
  },

  async setTransactionId(input) {
    const result = await primary()
      .update(schema.computeSignupCreditClaims)
      .set({ stripeBalanceTransactionId: input.transactionId })
      .where(
        and(
          eq(schema.computeSignupCreditClaims.workspaceId, input.workspaceId),
          eq(schema.computeSignupCreditClaims.cardFingerprint, input.cardFingerprint),
          eq(schema.computeSignupCreditClaims.workosUserId, input.workosUserId),
          eq(schema.computeSignupCreditClaims.attemptId, input.attemptId),
          isNull(schema.computeSignupCreditClaims.stripeBalanceTransactionId),
        ),
      );
    return affectedRows(result);
  },

  async deleteAttempt(claim) {
    await primary()
      .delete(schema.computeSignupCreditClaims)
      .where(
        and(
          eq(schema.computeSignupCreditClaims.attemptId, claim.attemptId),
          eq(schema.computeSignupCreditClaims.workspaceId, claim.workspaceId),
          eq(schema.computeSignupCreditClaims.cardFingerprint, claim.cardFingerprint),
          eq(schema.computeSignupCreditClaims.workosUserId, claim.workosUserId),
          isNull(schema.computeSignupCreditClaims.stripeBalanceTransactionId),
        ),
      );
  },
};
