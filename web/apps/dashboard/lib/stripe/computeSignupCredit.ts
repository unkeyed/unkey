import { randomUUID } from "node:crypto";
import { db } from "@/lib/db";
import { stripeEnv } from "@/lib/env";
import Stripe from "stripe";
import { z } from "zod";
import { COMPUTE_SIGNUP_CREDIT_CENTS } from "./computeSignupCreditAmount";
import {
  type ClaimIdentity,
  type ClaimKey,
  type ComputeSignupCreditClaim,
  deleteClaimAttempt,
  findClaimByWorkspace,
  findClaimConflicts,
  insertClaim,
  setClaimTransactionId,
  takeOverClaim,
} from "./computeSignupCreditClaims";

export { COMPUTE_SIGNUP_CREDIT_CENTS };
export const COMPUTE_SIGNUP_CREDIT_CURRENCY = "usd";
export const COMPUTE_SIGNUP_CREDIT_PURPOSE = "compute_signup";

/** A missing workspace on a customer newer than this is retried, not skipped. */
export const WORKSPACE_LINK_RETRY_SECONDS = 60 * 60;
export const UNFINISHED_CLAIM_STALE_MS = 60 * 60 * 1000;

const PURPOSE_METADATA_KEY = "unkey_credit";
const USER_METADATA_KEY = "workos_user_id";
const FINGERPRINT_PATTERN = /^[A-Za-z0-9]+$/;
const WORKOS_USER_PATTERN = /^user_[A-Za-z0-9]+$/;

const attachedPaymentMethodSchema = z.object({
  id: z.string().min(1),
  object: z.literal("payment_method"),
  customer: z.union([z.string(), z.object({ id: z.string() }).passthrough(), z.null()]).optional(),
});

export class ComputeSignupCreditRetryError extends Error {
  readonly reason: string;

  constructor(reason: string) {
    super(reason);
    this.name = "ComputeSignupCreditRetryError";
    this.reason = reason;
  }
}

export type ComputeSignupCreditResult =
  | { granted: true; transactionId: string; amountCents: number }
  | { granted: false; reason: string };

export type SignupWorkspaceResolution =
  | { status: "found"; workspaceId: string }
  | { status: "deleted" }
  | { status: "missing" };

export type ComputeSignupCreditInput = {
  workspaceId: string;
  customerId: string;
  fingerprint: string;
  workosUserId: string;
  nowMs: number;
};

type AttachedDeps = {
  nowSeconds?: number;
  resolveUserId?: (input: {
    customerId: string;
    paymentMethodId: string;
    workspaceId: string;
  }) => Promise<string | null>;
};

type SignupCredit = {
  id: string;
  metadata: Stripe.Metadata | null;
};

type SignupCreditIdentity = {
  cardFingerprint: string;
  workosUserId: string;
};

type SignupCreditLookup = (customerId: string) => Promise<SignupCredit | undefined>;

export function signupCardEligibility(
  fingerprint: string | null,
): { eligible: true; fingerprint: string } | { eligible: false; reason: string } {
  if (!fingerprint || !FINGERPRINT_PATTERN.test(fingerprint)) {
    return { eligible: false, reason: "card has no fingerprint" };
  }
  return { eligible: true, fingerprint };
}

export function computeSignupCreditIdempotencyKey(workspaceId: string): string {
  return `compute-signup-credit:${workspaceId}`;
}

export function signupWorkosUserId(raw: string | null | undefined): string | null {
  if (!raw || !WORKOS_USER_PATTERN.test(raw)) {
    return null;
  }
  return raw;
}

function isIdempotencyError(error: unknown): boolean {
  return error instanceof Stripe.errors.StripeIdempotencyError;
}

function customerIdOf(customer: Stripe.PaymentMethod["customer"]): string | null {
  if (!customer) {
    return null;
  }
  if (typeof customer === "string") {
    return customer;
  }
  if ("deleted" in customer && customer.deleted) {
    return null;
  }
  return customer.id;
}

function customerIdFromEvent(customer: string | { id: string } | null | undefined): string | null {
  if (!customer) {
    return null;
  }
  if (typeof customer === "string") {
    return customer;
  }
  return customer.id;
}

function isSignupCredit(metadata: Stripe.Metadata | null | undefined): boolean {
  return metadata?.[PURPOSE_METADATA_KEY] === COMPUTE_SIGNUP_CREDIT_PURPOSE;
}

function creditCoversRow(credit: SignupCredit, row: SignupCreditIdentity): boolean {
  const userId = signupWorkosUserId(credit.metadata?.[USER_METADATA_KEY]);
  const fingerprint = credit.metadata?.card_fingerprint;
  if (!userId || !fingerprint) {
    return false;
  }
  return userId === row.workosUserId && fingerprint === row.cardFingerprint;
}

async function existingSignupCredit(
  stripe: Stripe,
  customerId: string,
  identity: SignupCreditIdentity,
): Promise<SignupCredit | undefined> {
  for await (const transaction of stripe.customers.listBalanceTransactions(customerId, {
    limit: 100,
  })) {
    if (!isSignupCredit(transaction.metadata)) {
      continue;
    }
    const credit = { id: transaction.id, metadata: transaction.metadata };
    if (!creditCoversRow(credit, identity)) {
      continue;
    }
    return credit;
  }
  return undefined;
}

async function workspaceStatus(workspaceId: string): Promise<"active" | "deleted" | "missing"> {
  const workspace = await db.query.workspaces.findFirst({
    where: (table, { eq }) => eq(table.id, workspaceId),
    columns: { id: true, deletedAtM: true },
  });
  if (!workspace) {
    return "missing";
  }
  if (workspace.deletedAtM !== null) {
    return "deleted";
  }
  return "active";
}

export async function resolveSignupWorkspaceId(
  stripe: Stripe,
  customerId: string,
): Promise<SignupWorkspaceResolution> {
  const billing = await db.query.workspaceBilling.findFirst({
    where: (table, { eq }) => eq(table.stripeCustomerId, customerId),
    columns: { workspaceId: true },
    with: {
      workspace: { columns: { deletedAtM: true } },
    },
  });
  if (billing) {
    if (billing.workspace.deletedAtM !== null) {
      return { status: "deleted" };
    }
    return { status: "found", workspaceId: billing.workspaceId };
  }

  let sawDeleted = false;
  for await (const session of stripe.checkout.sessions.list({
    customer: customerId,
    limit: 100,
  })) {
    const workspaceId = session.client_reference_id;
    if (!workspaceId) {
      continue;
    }
    const status = await workspaceStatus(workspaceId);
    if (status === "active") {
      return { status: "found", workspaceId };
    }
    if (status === "deleted") {
      sawDeleted = true;
    }
  }

  const customer = await stripe.customers.retrieve(customerId);
  if (!customer.deleted) {
    const workspaceId = customer.metadata.workspace_id;
    if (workspaceId) {
      const status = await workspaceStatus(workspaceId);
      if (status === "active") {
        return { status: "found", workspaceId };
      }
      if (status === "deleted") {
        sawDeleted = true;
      }
    }
  }

  if (sawDeleted) {
    return { status: "deleted" };
  }
  return { status: "missing" };
}

type AcquireResult =
  | { status: "held"; claim: ClaimKey }
  | { status: "fingerprint_used" }
  | { status: "user_used" }
  | { status: "workspace_credited"; transactionId: string }
  | { status: "conflict" };

function createStoredNothing(error: unknown): boolean {
  if (!(error instanceof Stripe.errors.StripeInvalidRequestError)) {
    return false;
  }
  const status = error.statusCode;
  return (
    typeof status === "number" &&
    status >= 400 &&
    status < 500 &&
    error.rawType === "invalid_request_error"
  );
}

async function acquireClaim(
  identity: ClaimIdentity,
  customerId: string,
  nowMs: number,
  creditForCustomer: SignupCreditLookup,
): Promise<AcquireResult> {
  const { workspaceId, cardFingerprint: fingerprint, workosUserId } = identity;
  const attemptId = randomUUID();
  const inserted = await insertClaim({
    ...identity,
    stripeCustomerId: customerId,
    createdAt: nowMs,
    attemptId,
  });
  if (inserted === "inserted") {
    return { status: "held", claim: { ...identity, attemptId } };
  }

  const { byWorkspace, byFingerprint, byUser } = await findClaimConflicts(identity);
  if (!byWorkspace && !byFingerprint && !byUser) {
    throw new ComputeSignupCreditRetryError("signup credit claim conflicted but no row was found");
  }
  if (byUser?.stripeBalanceTransactionId && byUser.workspaceId !== workspaceId) {
    return { status: "user_used" };
  }
  if (byFingerprint?.stripeBalanceTransactionId && byFingerprint.workspaceId !== workspaceId) {
    return { status: "fingerprint_used" };
  }
  if (byWorkspace?.stripeBalanceTransactionId) {
    return {
      status: "workspace_credited",
      transactionId: byWorkspace.stripeBalanceTransactionId,
    };
  }

  const sameCard =
    byWorkspace?.cardFingerprint === fingerprint &&
    byWorkspace.workosUserId === workosUserId &&
    byWorkspace.stripeBalanceTransactionId === null
      ? byWorkspace
      : undefined;
  if (!sameCard) {
    console.error("Compute signup credit left unfinished for manual reconcile", {
      workspaceId,
      cardFingerprint: fingerprint,
      workosUserId,
      existingWorkspaceId: byWorkspace?.workspaceId ?? null,
      existingFingerprint: byFingerprint?.cardFingerprint ?? byUser?.cardFingerprint ?? null,
      existingUserId: byUser?.workosUserId ?? byWorkspace?.workosUserId ?? null,
      existingAttemptId: byWorkspace?.attemptId ?? byFingerprint?.attemptId ?? byUser?.attemptId,
    });
    return { status: "conflict" };
  }
  if (nowMs - sameCard.createdAt < UNFINISHED_CLAIM_STALE_MS) {
    throw new ComputeSignupCreditRetryError("signup credit claim is in progress");
  }

  const hidden = await creditForCustomer(sameCard.stripeCustomerId);
  const current = await findClaimByWorkspace(workspaceId);
  if (!sameTupleAttempt(current, sameCard, nowMs)) {
    throw new ComputeSignupCreditRetryError("signup credit claim is in progress");
  }
  if (hidden && creditCoversRow(hidden, current)) {
    await recordTransaction({ ...identity, attemptId: current.attemptId }, hidden.id);
    return { status: "workspace_credited", transactionId: hidden.id };
  }

  const held = { ...identity, attemptId: randomUUID() };
  const won = await takeOverClaim(current, {
    ...held,
    stripeCustomerId: customerId,
    createdAt: nowMs,
  });
  if (!won) {
    throw new ComputeSignupCreditRetryError("signup credit claim is in progress");
  }
  return { status: "held", claim: held };
}

function sameTupleAttempt(
  current: ComputeSignupCreditClaim | null,
  judged: ComputeSignupCreditClaim,
  nowMs: number,
): current is ComputeSignupCreditClaim {
  return (
    current !== null &&
    current.pk === judged.pk &&
    current.attemptId === judged.attemptId &&
    current.cardFingerprint === judged.cardFingerprint &&
    current.workosUserId === judged.workosUserId &&
    current.stripeBalanceTransactionId === null &&
    nowMs - current.createdAt >= UNFINISHED_CLAIM_STALE_MS
  );
}

async function recordTransaction(held: ClaimKey, transactionId: string): Promise<void> {
  if (await setClaimTransactionId(held, transactionId)) {
    return;
  }
  const row = await findClaimByWorkspace(held.workspaceId);
  if (
    row?.stripeBalanceTransactionId &&
    row.cardFingerprint === held.cardFingerprint &&
    row.workosUserId === held.workosUserId &&
    row.attemptId === held.attemptId
  ) {
    return;
  }
  console.error("Compute signup credit claim conflict while recording the balance transaction", {
    workspaceId: held.workspaceId,
    cardFingerprint: held.cardFingerprint,
    attemptId: held.attemptId,
    transactionId,
  });
  throw new ComputeSignupCreditRetryError(
    "signup credit claim conflict while recording the balance transaction",
  );
}

async function stampListedCredit(
  workspaceId: string,
  customerId: string,
  workosUserId: string,
  credit: SignupCredit,
  nowMs: number,
): Promise<void> {
  const fingerprint = credit.metadata?.card_fingerprint;
  if (!fingerprint || !FINGERPRINT_PATTERN.test(fingerprint)) {
    return;
  }
  const creditedUser = signupWorkosUserId(credit.metadata?.[USER_METADATA_KEY]) ?? workosUserId;
  const identity = { workspaceId, cardFingerprint: fingerprint, workosUserId: creditedUser };
  const { byWorkspace, byFingerprint, byUser } = await findClaimConflicts(identity);
  const sameRow =
    byWorkspace?.cardFingerprint === fingerprint && byWorkspace.workosUserId === creditedUser
      ? byWorkspace
      : null;
  if (sameRow) {
    if (sameRow.stripeBalanceTransactionId === null) {
      await recordTransaction({ ...identity, attemptId: sameRow.attemptId }, credit.id);
    }
    return;
  }
  if (byWorkspace || byFingerprint || byUser) {
    console.error("Compute signup credit left unfinished for manual reconcile", {
      workspaceId,
      cardFingerprint: fingerprint,
      workosUserId: creditedUser,
      existingAttemptId:
        byWorkspace?.attemptId ?? byFingerprint?.attemptId ?? byUser?.attemptId ?? null,
    });
    return;
  }
  const inserted = await insertClaim({
    ...identity,
    stripeCustomerId: customerId,
    stripeBalanceTransactionId: credit.id,
    createdAt: nowMs,
    attemptId: randomUUID(),
  });
  if (inserted !== "inserted") {
    console.error("Compute signup credit left unfinished for manual reconcile", identity);
  }
}

export async function grantComputeSignupCredit(
  stripe: Stripe,
  input: ComputeSignupCreditInput,
): Promise<ComputeSignupCreditResult> {
  const eligibility = signupCardEligibility(input.fingerprint);
  if (!eligibility.eligible) {
    return { granted: false, reason: eligibility.reason };
  }

  const customer = await stripe.customers.retrieve(input.customerId);

  const identity = {
    cardFingerprint: input.fingerprint,
    workosUserId: input.workosUserId,
  };
  const existing = await existingSignupCredit(stripe, input.customerId, identity);
  if (existing) {
    await stampListedCredit(
      input.workspaceId,
      input.customerId,
      input.workosUserId,
      existing,
      input.nowMs,
    );
    return { granted: false, reason: `already credited (${existing.id})` };
  }

  const acquired = await acquireClaim(
    {
      workspaceId: input.workspaceId,
      cardFingerprint: input.fingerprint,
      workosUserId: input.workosUserId,
    },
    input.customerId,
    input.nowMs,
    (customerId) => existingSignupCredit(stripe, customerId, identity),
  );
  if (acquired.status === "user_used") {
    return { granted: false, reason: "workos user already credited" };
  }
  if (acquired.status === "fingerprint_used") {
    return { granted: false, reason: "card fingerprint already used" };
  }
  if (acquired.status === "workspace_credited") {
    return {
      granted: false,
      reason: `already credited (${acquired.transactionId})`,
    };
  }
  if (acquired.status === "conflict") {
    return { granted: false, reason: "signup credit claim conflicts with another row" };
  }
  const held = acquired.claim;

  if (customer.deleted) {
    await deleteClaimAttempt(held);
    return { granted: false, reason: "customer deleted" };
  }

  let created: Stripe.CustomerBalanceTransaction;
  try {
    created = await stripe.customers.createBalanceTransaction(
      input.customerId,
      {
        amount: -COMPUTE_SIGNUP_CREDIT_CENTS,
        currency: COMPUTE_SIGNUP_CREDIT_CURRENCY,
        description: "Compute signup credit",
        metadata: {
          [PURPOSE_METADATA_KEY]: COMPUTE_SIGNUP_CREDIT_PURPOSE,
          workspace_id: input.workspaceId,
          card_fingerprint: input.fingerprint,
          [USER_METADATA_KEY]: input.workosUserId,
        },
      },
      { idempotencyKey: computeSignupCreditIdempotencyKey(input.workspaceId) },
    );
  } catch (error) {
    if (isIdempotencyError(error)) {
      const credit = await existingSignupCredit(stripe, input.customerId, held);
      if (!credit) {
        // Keep the row. Deleting it would free the unique keys, and after this
        // workspace key expires another customer can be paid a second -$5.
        throw new ComputeSignupCreditRetryError(
          "signup credit create conflicted and no balance transaction exists",
        );
      }
      await recordTransaction(held, credit.id);
      return { granted: false, reason: `already credited (${credit.id})` };
    }
    if (createStoredNothing(error)) {
      await deleteClaimAttempt(held);
    }
    throw error;
  }

  await recordTransaction(held, created.id);
  return {
    granted: true,
    transactionId: created.id,
    amountCents: COMPUTE_SIGNUP_CREDIT_CENTS,
  };
}

function expandableId(value: string | { id: string } | null | undefined): string | null {
  if (!value) {
    return null;
  }
  return typeof value === "string" ? value : value.id;
}

function paymentIntentCustomerId(intent: Stripe.PaymentIntent): string | null {
  const customer = intent.customer;
  if (!customer || (typeof customer === "object" && "deleted" in customer && customer.deleted)) {
    return null;
  }
  return expandableId(customer);
}

async function paymentMethodIdOnIntent(
  stripe: Stripe,
  intent: string | Stripe.PaymentIntent,
  customerId: string,
): Promise<string | null> {
  let resolved: Stripe.PaymentIntent;
  if (typeof intent === "string") {
    try {
      resolved = await stripe.paymentIntents.retrieve(intent);
    } catch (error) {
      if (
        error instanceof Stripe.errors.StripeInvalidRequestError &&
        error.code === "resource_missing"
      ) {
        console.warn("Compute signup credit skipped: payment intent is missing", {
          customerId,
          paymentIntentId: intent,
        });
        return null;
      }
      throw error;
    }
  } else {
    resolved = intent;
  }
  if (paymentIntentCustomerId(resolved) !== customerId) {
    return null;
  }
  return expandableId(resolved.payment_method);
}

async function invoiceChargesCard(
  stripe: Stripe,
  invoice: Stripe.Subscription["latest_invoice"],
  paymentMethodId: string,
  customerId: string,
): Promise<boolean> {
  if (!invoice || typeof invoice === "string") {
    return false;
  }
  if (expandableId(invoice.default_payment_method) === paymentMethodId) {
    return true;
  }
  for (const payment of invoice.payments?.data ?? []) {
    const intent = payment.payment?.payment_intent;
    if (!intent) {
      continue;
    }
    if ((await paymentMethodIdOnIntent(stripe, intent, customerId)) === paymentMethodId) {
      return true;
    }
  }
  return false;
}

async function subscriptionChargesCard(
  stripe: Stripe,
  subscription: Stripe.Subscription,
  paymentMethodId: string,
  customerId: string,
): Promise<boolean> {
  return (
    expandableId(subscription.default_payment_method) === paymentMethodId ||
    invoiceChargesCard(stripe, subscription.latest_invoice, paymentMethodId, customerId)
  );
}

async function userIdFromSetupIntent(
  stripe: Stripe,
  customerId: string,
  paymentMethodId: string,
): Promise<{ status: "found"; userId: string } | { status: "invalid" } | { status: "none" }> {
  let invalid = false;
  for await (const intent of stripe.setupIntents.list({
    customer: customerId,
    payment_method: paymentMethodId,
    limit: 100,
  })) {
    if (expandableId(intent.payment_method) !== paymentMethodId) {
      continue;
    }
    const raw = intent.metadata?.[USER_METADATA_KEY];
    if (!raw) {
      continue;
    }
    const userId = signupWorkosUserId(raw);
    if (userId) {
      return { status: "found", userId };
    }
    invalid = true;
  }
  return invalid ? { status: "invalid" } : { status: "none" };
}

async function userIdFromSubscriptionPaymentMethod(
  stripe: Stripe,
  customerId: string,
  paymentMethodId: string,
): Promise<string | null> {
  for await (const subscription of stripe.subscriptions.list({
    customer: customerId,
    status: "all",
    limit: 100,
    expand: ["data.latest_invoice.payments"],
  })) {
    if (!(await subscriptionChargesCard(stripe, subscription, paymentMethodId, customerId))) {
      continue;
    }
    const userId = signupWorkosUserId(subscription.metadata?.[USER_METADATA_KEY]);
    if (userId) {
      return userId;
    }
  }
  return null;
}

async function subscriptionHasPaymentMethod(
  stripe: Stripe,
  subscription: Stripe.Subscription,
  customerId: string,
): Promise<boolean> {
  if (expandableId(subscription.default_payment_method)) {
    return true;
  }
  const invoice = subscription.latest_invoice;
  if (!invoice || typeof invoice === "string") {
    return false;
  }
  if (expandableId(invoice.default_payment_method)) {
    return true;
  }
  for (const payment of invoice.payments?.data ?? []) {
    const intent = payment.payment?.payment_intent;
    if (!intent) {
      continue;
    }
    if (await paymentMethodIdOnIntent(stripe, intent, customerId)) {
      return true;
    }
  }
  return false;
}

function sessionLifeCoversCard(
  sessionCreated: number,
  sessionExpiresAt: number | null | undefined,
  paymentMethodCreated: number,
): boolean {
  if (sessionExpiresAt == null) {
    return false;
  }
  return sessionCreated <= paymentMethodCreated && paymentMethodCreated <= sessionExpiresAt;
}

async function openSubscriptionCheckoutCoversCard(
  stripe: Stripe,
  customerId: string,
  workspaceId: string,
  paymentMethodCreated: number,
): Promise<boolean> {
  for await (const session of stripe.checkout.sessions.list({
    customer: customerId,
    status: "open",
    limit: 100,
  })) {
    if (session.status !== "open") {
      continue;
    }
    if (session.mode !== "subscription" && !session.subscription) {
      continue;
    }
    if (session.client_reference_id !== workspaceId) {
      continue;
    }
    if (!sessionLifeCoversCard(session.created, session.expires_at, paymentMethodCreated)) {
      continue;
    }
    const subscriptionId = expandableId(session.subscription);
    if (!subscriptionId) {
      return true;
    }
    const subscription = await stripe.subscriptions.retrieve(subscriptionId, {
      expand: ["latest_invoice.payments"],
    });
    if (!(await subscriptionHasPaymentMethod(stripe, subscription, customerId))) {
      return true;
    }
  }
  return false;
}

export async function resolveSignupWorkosUserId(
  stripe: Stripe,
  customerId: string,
  paymentMethodId: string,
  workspaceId: string,
  paymentMethodCreated?: number,
): Promise<string | null> {
  const fromSetup = await userIdFromSetupIntent(stripe, customerId, paymentMethodId);
  if (fromSetup.status === "found") {
    return fromSetup.userId;
  }
  const fromSubscription = await userIdFromSubscriptionPaymentMethod(
    stripe,
    customerId,
    paymentMethodId,
  );
  if (fromSubscription) {
    return fromSubscription;
  }
  if (
    paymentMethodCreated !== undefined &&
    (await openSubscriptionCheckoutCoversCard(
      stripe,
      customerId,
      workspaceId,
      paymentMethodCreated,
    ))
  ) {
    throw new ComputeSignupCreditRetryError("subscription checkout has no payment method yet");
  }
  return null;
}

export async function handlePaymentMethodAttached(
  stripe: Stripe,
  payload: unknown,
  deps: AttachedDeps = {},
): Promise<ComputeSignupCreditResult> {
  const parsed = attachedPaymentMethodSchema.safeParse(payload);
  if (!parsed.success) {
    return { granted: false, reason: "invalid payment method payload" };
  }

  const paymentMethod = await stripe.paymentMethods.retrieve(parsed.data.id);
  const customerId =
    customerIdOf(paymentMethod.customer) ?? customerIdFromEvent(parsed.data.customer);
  if (!customerId) {
    return { granted: false, reason: "payment method has no customer" };
  }
  if (paymentMethod.type !== "card" || !paymentMethod.card) {
    return { granted: false, reason: "not a card" };
  }

  const eligibility = signupCardEligibility(paymentMethod.card.fingerprint ?? null);
  if (!eligibility.eligible) {
    return { granted: false, reason: eligibility.reason };
  }

  if (!stripeEnv()) {
    return { granted: false, reason: "stripe is not configured" };
  }

  const workspace = await resolveSignupWorkspaceId(stripe, customerId);
  if (workspace.status === "deleted") {
    return { granted: false, reason: "workspace deleted" };
  }
  if (workspace.status === "missing") {
    const customer = await stripe.customers.retrieve(customerId);
    if (customer.deleted) {
      return { granted: false, reason: "customer deleted" };
    }
    const nowSeconds = deps.nowSeconds ?? Math.floor(Date.now() / 1000);
    if (nowSeconds - customer.created <= WORKSPACE_LINK_RETRY_SECONDS) {
      throw new ComputeSignupCreditRetryError("workspace not linked yet");
    }
    return { granted: false, reason: "workspace not found" };
  }

  const nowSeconds = deps.nowSeconds ?? Math.floor(Date.now() / 1000);
  const workosUserId = deps.resolveUserId
    ? await deps.resolveUserId({
        customerId,
        paymentMethodId: paymentMethod.id,
        workspaceId: workspace.workspaceId,
      })
    : await resolveSignupWorkosUserId(
        stripe,
        customerId,
        paymentMethod.id,
        workspace.workspaceId,
        paymentMethod.created,
      );
  if (!workosUserId) {
    console.info("Compute signup credit skipped: card has no exact workos user", {
      workspaceId: workspace.workspaceId,
      customerId,
      paymentMethodId: paymentMethod.id,
    });
    return { granted: false, reason: "signup user not found" };
  }

  return grantComputeSignupCredit(stripe, {
    workspaceId: workspace.workspaceId,
    customerId,
    fingerprint: eligibility.fingerprint,
    workosUserId,
    nowMs: nowSeconds * 1000,
  });
}
