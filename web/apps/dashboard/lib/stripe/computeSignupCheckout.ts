import { routes } from "@/lib/navigation/routes";
import type Stripe from "stripe";
import {
  type ComputeSignupCreditResult,
  ComputeSignupCreditRetryError,
  grantComputeSignupCredit,
  signupCardEligibility,
  signupWorkosUserId,
} from "./computeSignupCredit";
import {
  type ComputeSignupCreditClaimStore,
  drizzleComputeSignupCreditClaimStore,
} from "./computeSignupCreditClaims";
import type { DeployPlan } from "./deployPlan";

export class ComputeSignupCheckoutError extends Error {
  constructor(message: string) {
    super(message);
    this.name = "ComputeSignupCheckoutError";
  }
}

export type DeployCheckoutCredit =
  | { step: "collect_card" }
  | {
      step: "subscribe";
      customerId?: string;
      credit: ComputeSignupCreditResult | null;
      announceSignupCredit: boolean;
    };

type GrantCard = {
  fingerprint: string;
};

export function deployCheckoutCustomerId(input: {
  existingCustomerId?: string;
  devClockedCustomerId?: string;
  setupSessionId?: string;
}): string | undefined {
  if (input.setupSessionId) {
    return input.existingCustomerId;
  }
  return input.existingCustomerId ?? input.devClockedCustomerId;
}

export function deployCheckoutSubmitMessage(input: {
  announceSignupCredit: boolean;
  signupCredit: string;
  planFee?: string;
}): string {
  if (!input.planFee) {
    return "";
  }
  return `Your plan fee is matched by usage credits: ${input.planFee} each month, and a prorated first charge is matched by the same amount in credits.`;
}

export function deployCardSetupSuccessUrl(input: {
  baseUrl: string;
  workspaceSlug: string;
  plan: DeployPlan;
  from?: "create" | "banner" | "billing" | "deploy";
  returnTo?: string;
}): string {
  const path = routes.settings.stripe.checkout({
    workspaceSlug: input.workspaceSlug,
    intent: "deploy",
    plan: input.plan,
    from: input.from,
    returnTo: input.returnTo,
  });
  return `${input.baseUrl}${path}&setup_session_id={CHECKOUT_SESSION_ID}`;
}

function customerIdFrom(customer: Stripe.Checkout.Session["customer"]): string | null {
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

function defaultPaymentMethodId(customer: Stripe.Customer | Stripe.DeletedCustomer): string | null {
  if (customer.deleted) {
    return null;
  }
  const method = customer.invoice_settings?.default_payment_method;
  if (!method) {
    return null;
  }
  return typeof method === "string" ? method : method.id;
}

function grantCard(card: Stripe.PaymentMethod.Card): GrantCard {
  return { fingerprint: card.fingerprint ?? "" };
}

async function eligibleSavedCard(stripe: Stripe, customerId: string): Promise<GrantCard | null> {
  const customer = await stripe.customers.retrieve(customerId);
  if (customer.deleted) {
    return null;
  }
  const defaultId = defaultPaymentMethodId(customer);
  const ordered: Stripe.PaymentMethod[] = [];
  for await (const method of stripe.paymentMethods.list({
    customer: customerId,
    type: "card",
    limit: 100,
  })) {
    ordered.push(method);
  }
  if (defaultId) {
    ordered.sort((left, right) => {
      if (left.id === defaultId) {
        return -1;
      }
      if (right.id === defaultId) {
        return 1;
      }
      return 0;
    });
  }
  for (const method of ordered) {
    if (method.type !== "card" || !method.card) {
      continue;
    }
    const eligibility = signupCardEligibility(method.card.fingerprint ?? null);
    if (!eligibility.eligible) {
      continue;
    }
    return { fingerprint: eligibility.fingerprint };
  }
  return null;
}

async function cardFromSetupSession(
  stripe: Stripe,
  setupSessionId: string,
  workspaceId: string,
  expectedCustomerId: string | undefined,
): Promise<{ customerId: string; card: GrantCard; intentUserId: string | null }> {
  const session = await stripe.checkout.sessions.retrieve(setupSessionId, {
    expand: ["setup_intent.payment_method"],
  });
  if (session.mode !== "setup" || session.client_reference_id !== workspaceId) {
    throw new ComputeSignupCheckoutError("This card setup session is for a different workspace.");
  }
  if (session.status !== "complete") {
    throw new ComputeSignupCheckoutError("Card setup did not finish. Start checkout again.");
  }
  const customerId = customerIdFrom(session.customer);
  if (!customerId) {
    throw new ComputeSignupCheckoutError("Card setup did not finish. Start checkout again.");
  }
  if (expectedCustomerId && expectedCustomerId !== customerId) {
    throw new ComputeSignupCheckoutError("This card setup session is for a different customer.");
  }
  const intent = session.setup_intent;
  if (!intent || typeof intent === "string") {
    throw new ComputeSignupCheckoutError("Card setup did not finish. Start checkout again.");
  }
  const rawMethod = intent.payment_method;
  if (!rawMethod) {
    throw new ComputeSignupCheckoutError("Card setup did not finish. Start checkout again.");
  }
  const paymentMethod =
    typeof rawMethod === "string" ? await stripe.paymentMethods.retrieve(rawMethod) : rawMethod;
  if (paymentMethod.type !== "card" || !paymentMethod.card) {
    throw new ComputeSignupCheckoutError("Checkout needs a card.");
  }
  const rawIntentUser = intent.metadata?.workos_user_id;
  return {
    customerId,
    card: grantCard(paymentMethod.card),
    intentUserId: typeof rawIntentUser === "string" ? rawIntentUser : null,
  };
}

function announceCredit(credit: ComputeSignupCreditResult | null): boolean {
  return credit?.granted === true;
}

function subscribeWithoutCredit(customerId: string | undefined): DeployCheckoutCredit {
  return {
    step: "subscribe",
    ...(customerId ? { customerId } : {}),
    credit: null,
    announceSignupCredit: false,
  };
}

async function grantForCard(
  stripe: Stripe,
  store: ComputeSignupCreditClaimStore,
  input: {
    workspaceId: string;
    customerId: string;
    workosUserId: string;
    nowMs: number;
    card: GrantCard;
  },
): Promise<ComputeSignupCreditResult> {
  return grantComputeSignupCredit(
    stripe,
    {
      workspaceId: input.workspaceId,
      customerId: input.customerId,
      fingerprint: input.card.fingerprint,
      workosUserId: input.workosUserId,
      nowMs: input.nowMs,
    },
    store,
  );
}

async function grantOrContinue(
  stripe: Stripe,
  store: ComputeSignupCreditClaimStore,
  input: {
    workspaceId: string;
    customerId: string;
    workosUserId: string;
    nowMs: number;
    card: GrantCard;
  },
): Promise<ComputeSignupCreditResult | null> {
  try {
    return await grantForCard(stripe, store, input);
  } catch (error) {
    if (error instanceof ComputeSignupCreditRetryError) {
      throw error;
    }
    console.error("Compute signup credit grant failed; continuing checkout", {
      workspaceId: input.workspaceId,
      customerId: input.customerId,
      error,
    });
    return null;
  }
}

export async function prepareDeployCheckoutCredit(
  stripe: Stripe,
  input: {
    workspaceId: string;
    workosUserId: string;
    customerId?: string;
    setupSessionId?: string;
    nowMs: number;
    store?: ComputeSignupCreditClaimStore;
  },
): Promise<DeployCheckoutCredit> {
  const payer = signupWorkosUserId(input.workosUserId);
  const store = input.store ?? drizzleComputeSignupCreditClaimStore;
  const claim = await store.findByWorkspace(input.workspaceId);
  if (claim?.stripeBalanceTransactionId) {
    return {
      step: "subscribe",
      customerId: claim.stripeCustomerId,
      credit: null,
      announceSignupCredit: false,
    };
  }
  if (!payer) {
    if (!input.setupSessionId) {
      return subscribeWithoutCredit(input.customerId);
    }
    const setup = await cardFromSetupSession(
      stripe,
      input.setupSessionId,
      input.workspaceId,
      input.customerId,
    );
    return subscribeWithoutCredit(setup.customerId);
  }

  if (input.setupSessionId) {
    const setup = await cardFromSetupSession(
      stripe,
      input.setupSessionId,
      input.workspaceId,
      input.customerId,
    );
    const intentPayer = signupWorkosUserId(setup.intentUserId);
    if (!intentPayer || intentPayer !== payer) {
      return subscribeWithoutCredit(setup.customerId);
    }
    const credit = await grantOrContinue(stripe, store, {
      workspaceId: input.workspaceId,
      customerId: setup.customerId,
      workosUserId: intentPayer,
      nowMs: input.nowMs,
      card: setup.card,
    });
    return {
      step: "subscribe",
      customerId: setup.customerId,
      credit,
      announceSignupCredit: announceCredit(credit),
    };
  }

  if (!input.customerId) {
    return { step: "collect_card" };
  }
  const saved = await eligibleSavedCard(stripe, input.customerId);
  if (!saved) {
    return { step: "collect_card" };
  }
  // The balance is customer-level, so when setup is skipped the paying card
  // may differ from the graded card. James accepts this.
  const credit = await grantOrContinue(stripe, store, {
    workspaceId: input.workspaceId,
    customerId: input.customerId,
    workosUserId: payer,
    nowMs: input.nowMs,
    card: saved,
  });
  return {
    step: "subscribe",
    customerId: input.customerId,
    credit,
    announceSignupCredit: announceCredit(credit),
  };
}
